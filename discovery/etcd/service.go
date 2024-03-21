package etcd

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/util/etd"
	"go.etcd.io/etcd/api/v3/mvccpb"
	etcd "go.etcd.io/etcd/client/v3"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	RegTtl           int64 = 5
	_RegRoot               = ""
	_RegSvcIdxOffset       = 1
	_SvcLeaseId      int64
)

func SetRegRoot(root string) {
	_RegRoot = root
	_RegSvcIdxOffset = strings.Count(RegSvcPrefix(), ".")
}

func RegSvcPrefix() string {
	return _RegRoot + "node.info"
}

func RegLockPrefix() string {
	return _RegRoot + "node.lock"
}

func RegisterService() {
	//退出前注销
	kiwi.BeforeExitFn("service unregister", unregisterSvc)
	registerSvc()
	go func() {
		svcWatch := etd.Client().Watch(util.Ctx(), RegSvcPrefix(), etcd.WithPrefix())
		for {
			select {
			case <-util.Ctx().Done():
				return
			case info := <-svcWatch:
				rcvSvcEvent(info)
			}
		}
	}()
}

func registerSvc() {
	svcSlc := make([]*kiwi.NodeMeta, 0, 8)
	err := etd.Lock(RegLockPrefix(), func() *util.Err {
		nodeIdMap := make(map[int64]struct{}, 8)
		_, e := etd.Get(RegSvcPrefix(), func(key string, bytes []byte) bool {
			var si kiwi.NodeMeta
			err := util.JsonUnmarshal(bytes, &si)
			if err != nil {
				return true
			}
			nodeIdMap[si.NodeId] = struct{}{}
			svcSlc = append(svcSlc, &si)
			return true
		}, etcd.WithPrefix())
		if e != nil {
			return util.WrapErr(util.EcEtcdErr, e)
		}
		nodeId := int64(0)
		info := kiwi.GetNodeMeta()
		for ; nodeId < 1024; nodeId++ {
			if _, ok := nodeIdMap[nodeId]; !ok {
				info.SetSvcId(nodeId)
				break
			}
		}
		if nodeId == 0 {
			return util.NewErr(util.EcServiceErr, util.M{
				"error": "too much service node",
			})
		}

		bytes, _ := util.JsonMarshal(info)
		str := string(bytes)
		key := getRegSvcKey(info.Svc, info.NodeId)
		leaseId, err := etd.PutWithTtl(key, str, RegTtl)
		if err != nil {
			return err
		}
		atomic.StoreInt64(&_SvcLeaseId, leaseId)
		kiwi.Info("register service success", util.M{
			"node":         leaseId,
			"lease id":     leaseId,
			"service info": info,
		})

		return nil
	})
	if err != nil {
		kiwi.Error(err)
		return
	}

	for _, si := range svcSlc {
		kiwi.Node().Connect(si.Ip, si.Port, si.Svc, si.SvcId, si.Ver, si.Data)
	}
}

func unregisterSvc() {
	id := atomic.SwapInt64(&_SvcLeaseId, 0)
	if id == 0 {
		return
	}
	kiwi.Info("unregister service", util.M{
		"info": kiwi.GetNodeMeta(),
	})
	_ = etd.Revoke(id)
}

func rcvSvcEvent(res etcd.WatchResponse) {
	for _, event := range res.Events {
		switch event.Type {
		case mvccpb.PUT:
			var si kiwi.NodeMeta
			err := util.JsonUnmarshal(event.Kv.Value, &si)
			if err != nil {
				kiwi.Error(err)
				return
			}
			if si.SvcId == kiwi.GetNodeMeta().NodeId {
				return
			}
			kiwi.Node().Connect(si.Ip, si.Port, si.Svc, si.SvcId, si.Ver, si.Data)
		case mvccpb.DELETE:
			key := string(event.Kv.Key)
			svc, id, err := splitRegSvcKey(key)
			if err != nil {
				kiwi.Error(err)
				return
			}
			if id == kiwi.GetNodeMeta().NodeId {
				return
			}
			kiwi.Node().Disconnect(svc, id)
		}
	}
}

func getRegSvcKey(svc kiwi.TSvc, id int64) string {
	return fmt.Sprintf("%s.%d.%d", RegSvcPrefix(), svc, id)
}

func splitRegSvcKey(key string) (svc kiwi.TSvc, id int64, err *util.Err) {
	slc := strings.Split(key, ".")
	l := len(slc)
	if l != _RegSvcIdxOffset+3 {
		err = util.NewErr(util.EcEtcdErr, util.M{
			"key": key,
		})
		return
	}
	svci, e := strconv.Atoi(slc[_RegSvcIdxOffset+1])
	if e != nil {
		err = util.WrapErr(util.EcParseErr, e)
		return
	}
	id, e = strconv.ParseInt(slc[_RegSvcIdxOffset+2], 10, 64)
	if e != nil {
		err = util.WrapErr(util.EcParseErr, e)
		return
	}
	svc = kiwi.TSvc(svci)
	return
}
