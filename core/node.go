package core

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/network"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"math/rand"
	"net"
)

type (
	NodeOption func(opt *nodeOption)
	nodeOption struct {
		ip       string
		port     int
		connType NodeConnType
		selector NodeDialerSelector
	}
	NodeConnType uint8
)

type NodeDialerSelector func(set *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer]) (kiwi.INodeDialer, *util.Err)

const (
	Tcp NodeConnType = iota
	Udp
)

func NodeType(t NodeConnType) NodeOption {
	return func(opt *nodeOption) {
		opt.connType = t
	}
}

func NodeIp(ip string) NodeOption {
	return func(opt *nodeOption) {
		opt.ip = ip
	}
}

func NodePort(port int) NodeOption {
	return func(opt *nodeOption) {
		opt.port = port
	}
}

func NodeSelector(selector NodeDialerSelector) NodeOption {
	return func(opt *nodeOption) {
		opt.selector = selector
	}
}

func NewNode(opts ...NodeOption) kiwi.INode {
	opt := &nodeOption{
		connType: Tcp,
		selector: func(set *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer]) (kiwi.INodeDialer, *util.Err) {
			i := rand.Intn(set.Count())
			dialer, _ := set.GetWithIdx(i)
			return dialer, nil
		},
	}
	for _, o := range opts {
		o(opt)
	}
	n := &node{
		option: opt,
		svcToDialer: ds.NewKSet2[kiwi.TSvc, int64, kiwi.INodeDialer](8, func(dialer kiwi.INodeDialer) int64 {
			return dialer.NodeId()
		}),
		idToDialer: ds.NewKSet[int64, kiwi.INodeDialer](16, func(dialer kiwi.INodeDialer) int64 {
			return dialer.NodeId()
		}),
		codeToWatchers: make(map[kiwi.TCode]map[int64]util.M),
	}
	ip, err := util.CheckLocalIp(opt.ip)
	if err != nil {
		kiwi.Fatal(err)
	}
	opt.ip = ip
	return n
}

type node struct {
	option         *nodeOption
	worker         *worker.Worker
	svcToDialer    *ds.KSet2[kiwi.TSvc, int64, kiwi.INodeDialer]
	idToDialer     *ds.KSet[int64, kiwi.INodeDialer]
	listener       kiwi.IListener
	codeToWatchers map[kiwi.TCode]map[int64]util.M //本机方法的远程监听者
	watcherToCodes map[int64][]kiwi.TCode          //远程机监听的方法
}

func (n *node) Init() {
	n.worker = worker.NewWorker(512, n.processor)
	addr := fmt.Sprintf("%s:%d", n.option.ip, n.option.port)
	var connType string
	switch n.option.connType {
	case Tcp:
		connType = "tcp"
		n.listener = network.NewTcpListener(addr, n.onAddTcpConn)
	case Udp:
		connType = "tcp"
		n.listener = network.NewUdpListener(addr, n.onAddUdpConn)
	}

	err := n.listener.Start()
	if err != nil {
		kiwi.Error3(util.EcListenErr, err)
	}
	port := n.listener.Port()
	meta := kiwi.GetNodeMeta()
	meta.Ip = n.option.ip
	meta.Port = port
	kiwi.Info("node listen", util.M{
		"type": connType,
		"meta": meta,
	})

	n.worker.Start()
}

func (n *node) Ip() string {
	return n.option.ip
}

func (n *node) Port() int {
	return n.listener.Port()
}

func (n *node) Connect(meta kiwi.SvcMeta) {
	n.worker.Push(nodeJobConnect{meta})
}

func (n *node) Disconnect(svc kiwi.TSvc, nodeId int64) {
	n.worker.Push(nodeJobDisconnect{svc, nodeId})
}

func (n *node) onConnected(dialer *nodeDialer) {
	n.worker.Push(nodeJobConnected{dialer})
}

func (n *node) onDisconnected(dialer *nodeDialer, err *util.Err) {
	n.worker.Push(nodeJobDisconnected{dialer, err})
}

func (n *node) pushSelf(pus kiwi.ISndPush) {
	pkt := NewRcvPusPkt()
	msg := pus.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdPush, pus.Tid(), pus.Head(), pus.Json(), pus.Msg())
	} else {
		err := pkt.InitWithBytes(HdPush, pus.Tid(), pus.Head(), pus.Json(), pus.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnPush(pkt)
}

func (n *node) Push(pus kiwi.ISndPush) {
	n.worker.Push(nodeJobPus{pus})
}

func (n *node) PushNode(nodeId int64, pus kiwi.ISndPush) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.pushSelf(pus)
		return
	}
	n.worker.Push(nodeJobPusNode{nodeId, pus})
}

func (n *node) requestSelf(req kiwi.ISndRequest) {
	pkt := NewRcvReqPkt()
	msg := req.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdRequest, req.Tid(), req.Head(), req.Json(), req.Msg())
	} else {
		err := pkt.InitWithBytes(HdRequest, req.Tid(), req.Head(), req.Json(), req.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnRequest(pkt)
}

func (n *node) Request(req kiwi.ISndRequest) {
	n.worker.Push(nodeJobReq{req})
}

func (n *node) RequestNode(nodeId int64, req kiwi.ISndRequest) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.requestSelf(req)
		return
	}
	n.worker.Push(nodeJobReqNode{nodeId, req})
}

func (n *node) Notify(ntc kiwi.ISndNotice, filter util.MToBool) {
	n.worker.Push(nodeJobSendNotice{ntc, filter})

	var pkt *RcvNtcPkt
	for _, service := range AllService() {
		if service.HasNoticeWatcher(ntc.Svc(), ntc.Code()) {
			if filter == nil || filter(service.Meta()) {
				if pkt == nil {
					pkt = NewRcvNtfPkt()
					if ntc.Msg() != nil {
						pkt.InitWithMsg(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Msg())
					} else {
						err := pkt.InitWithBytes(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Payload())
						if err != nil {
							kiwi.Error(err)
							return
						}
					}
				}
				service.OnNotice(pkt)
			}
		}
	}
}

func (n *node) NotifyOne(ntc kiwi.ISndNotice, filter util.MToBool) {
	for _, service := range AllService() {
		if service.HasNoticeWatcher(ntc.Svc(), ntc.Code()) {
			if filter == nil || filter(service.Meta()) {
				pkt := NewRcvNtfPkt()
				if ntc.Msg() != nil {
					pkt.InitWithMsg(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Msg())
				} else {
					err := pkt.InitWithBytes(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Payload())
					if err != nil {
						kiwi.Error(err)
						return
					}
				}
				service.OnNotice(pkt)
				return
			}
		}
	}
	n.worker.Push(nodeJobSendNoticeOne{ntc, filter})
}

func (n *node) ReceiveWatchNotice(nodeId int64, codes []kiwi.TCode, meta util.M) {
	n.worker.Push(nodeJobWatchNotice{nodeId, codes, meta})
}

func (n *node) SendToNode(nodeId int64, bytes []byte, fnErr util.FnErr) {
	n.worker.Push(nodeJobSendBytes{nodeId, bytes, fnErr})
}

func (n *node) onAddTcpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	agent := network.NewTcpAgent(addr, n.receive,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentMode(kiwi.AgentR),
		kiwi.AgentDeadline(30),
	)
	agent.Start(util.Ctx(), conn)
}

func (n *node) onAddUdpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	agent := network.NewUdpAgent(addr, n.receive,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentMode(kiwi.AgentR),
		kiwi.AgentDeadline(30),
	)
	agent.Start(util.Ctx(), conn)
}

func (n *node) createDialer(name, addr string) kiwi.IDialer {
	switch n.option.connType {
	case Tcp:
		return network.NewTcpDialer(name, addr, n.receive, kiwi.AgentMode(kiwi.AgentW))
	case Udp:
		return network.NewUdpDialer(name, addr, n.receive, kiwi.AgentMode(kiwi.AgentW))
	default:
		kiwi.Fatal2(util.EcParamsErr, util.M{
			"conn type": n.option.connType,
		})
		return nil
	}
}

func (n *node) processor(data any) {
	switch d := data.(type) {
	case nodeJobConnect:
		if n.idToDialer.Has(d.meta.NodeId) {
			return
		}
		kiwi.Info("connect service", util.M{
			"ip":      d.meta.Ip,
			"port":    d.meta.Port,
			"svc":     d.meta.Svc,
			"node id": d.meta.NodeId,
			"ver":     d.meta.Ver,
		})
		name := fmt.Sprintf("%d_%d", d.meta.Svc, d.meta.NodeId)
		addr := fmt.Sprintf("%s:%d", d.meta.Ip, d.meta.Port)
		dialer := n.createDialer(name, addr)
		newNodeDialer(dialer, d.meta.Svc, d.meta.NodeId, d.meta.Ver, n.onConnected, n.onDisconnected).connect()
	case nodeJobConnected:
		set, _ := n.svcToDialer.GetOrNew(d.dialer.svc, func() *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer] {
			return ds.NewSet2Item[kiwi.TSvc, int64, kiwi.INodeDialer](d.dialer.svc, 2, func(dialer kiwi.INodeDialer) int64 {
				return dialer.NodeId()
			})
		})
		old := set.Set(d.dialer)
		if old != nil {
			kiwi.Error2(util.EcExist, util.M{
				"node id": d.dialer.nodeId,
			})
		}
		_ = n.idToDialer.Set(d.dialer)
		kiwi.Info("node connected", util.M{
			"svc":     d.dialer.svc,
			"ver":     d.dialer.ver,
			"node id": d.dialer.nodeId,
		})
		//发送消息监听
		var codes []kiwi.TCode
		for _, service := range AllService() {
			c, ok := service.GetWatchCodes(d.dialer.Svc())
			if ok {
				codes = append(codes, c...)
			}
		}
		if len(codes) > 0 {
			nodeMeta := kiwi.GetNodeMeta()
			bytes := kiwi.Packer().PackWatchNotify(nodeMeta.NodeId, codes, nodeMeta.Data)
			d.dialer.Send(0, bytes)
		}
		kiwi.DispatchEvent(kiwi.Evt_Svc_Connected, &kiwi.EvtSvcConnected{
			Svc: d.dialer.svc,
			Id:  d.dialer.nodeId,
		})
	case nodeJobDisconnected:
		set, ok := n.svcToDialer.Get(d.dialer.svc)
		if !ok {
			return
		}
		nodeId := d.dialer.nodeId
		_, ok = set.Del(nodeId)
		if !ok {
			return
		}
		_, _ = n.idToDialer.Del(nodeId)
		codes, ok := n.watcherToCodes[nodeId]
		if ok {
			for _, code := range codes {
				m, ok := n.codeToWatchers[code]
				if ok {
					delete(m, nodeId)
				}
			}
		}
		kiwi.Info("dialer disconnected", util.M{
			"error":   d.err,
			"svc":     d.dialer.svc,
			"node id": d.dialer.nodeId,
		})
		kiwi.DispatchEvent(kiwi.Evt_Svc_Disonnected, &kiwi.EvtSvcDisconnected{
			Svc: d.dialer.svc,
			Id:  d.dialer.nodeId,
		})
	case nodeJobDisconnect:
		set, ok := n.svcToDialer.Get(d.svc)
		if !ok {
			return
		}
		dialer, ok := set.Get(d.nodeId)
		if !ok {
			return
		}
		kiwi.Info("disconnect service", util.M{
			"service": dialer.Svc(),
			"node id": dialer.NodeId(),
		})
		dialer.Dialer().Agent().Dispose()
	case nodeJobSendNotice:
		tid := d.notice.Tid()
		bytes, err := kiwi.Packer().PackNotify(tid, d.notice)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		m, ok := n.codeToWatchers[d.notice.Code()]
		if !ok {
			return
		}
		for nodeId, meta := range m {
			if d.filter == nil || d.filter(meta) {
				dialer, ok := n.idToDialer.Get(nodeId)
				if ok {
					dialer.Send(tid, util.CopyBytes(bytes))
				} else {
					delete(m, nodeId)
				}
			}
		}
	case nodeJobSendNoticeOne:
		tid := d.notice.Tid()
		bytes, err := kiwi.Packer().PackNotify(tid, d.notice)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		m, ok := n.codeToWatchers[d.notice.Code()]
		if !ok {
			return
		}
		for nodeId, meta := range m {
			if d.filter == nil || d.filter(meta) {
				dialer, ok := n.idToDialer.Get(nodeId)
				if ok {
					dialer.Send(tid, util.CopyBytes(bytes))
					break
				} else {
					delete(m, nodeId)
				}
			}
		}
	case nodeJobWatchNotice:
		_, ok := n.idToDialer.Get(d.nodeId)
		if !ok {
			kiwi.Error2(util.EcNotExist, util.M{
				"node id": d.nodeId,
			})
			return
		}
		n.watcherToCodes[d.nodeId] = d.codes
		for _, code := range d.codes {
			m, ok := n.codeToWatchers[code]
			if ok {
				m[d.nodeId] = d.meta
			} else {
				n.codeToWatchers[code] = map[int64]util.M{
					d.nodeId: d.meta,
				}
			}
		}
	case nodeJobPus:
		set, ok := n.svcToDialer.Get(d.pus.Svc())
		if !ok || set.Count() == 0 {
			ok = HasService(d.pus.Svc())
			if !ok {
				kiwi.Error2(util.EcNotExist, util.M{
					"svc": d.pus.Svc(),
				})
				return
			}
			n.pushSelf(d.pus)
			return
		}
		tid := d.pus.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, d.pus)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		if set.Count() == 1 {
			dialer, _ := set.GetWithIdx(0)
			dialer.Send(tid, bytes)
			return
		}
		dialer, err := n.option.selector(set)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		dialer.Send(tid, bytes)
	case nodeJobPusNode:
		tid := d.pus.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, d.pus)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(tid, d.nodeId, bytes)
	case nodeJobReq:
		set, ok := n.svcToDialer.Get(d.req.Svc())
		if !ok || set.Count() == 0 {
			ok = HasService(d.req.Svc())
			if !ok {
				kiwi.Error2(util.EcNotExist, util.M{
					"svc": d.req.Svc(),
				})
				return
			}
			n.requestSelf(d.req)
			return
		}
		tid := d.req.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, d.req)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		if set.Count() == 1 {
			dialer, _ := set.GetWithIdx(0)
			dialer.Send(tid, bytes)
			return
		}
		dialer, err := n.option.selector(set)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		dialer.Send(tid, bytes)
	case nodeJobReqNode:
		tid := d.req.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, d.req)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(tid, d.nodeId, bytes)
	case nodeJobSendBytes:
		n.sendToNode(0, d.nodeId, d.payload)
	}
}

func (n *node) sendToNode(tid, nodeId int64, bytes []byte) {
	dialer, ok := n.idToDialer.Get(nodeId)
	if !ok {
		if tid > 0 {
			kiwi.TE2(tid, util.EcNotExist, util.M{
				"id": nodeId,
			})
		} else {
			kiwi.Error2(util.EcNotExist, util.M{
				"id": nodeId,
			})
		}
		return
	}
	dialer.Send(tid, bytes)
}

func (n *node) receive(agent kiwi.IAgent, bytes []byte) {
	switch bytes[0] {
	case HdPush:
		n.onPush(agent, bytes)
	case HdRequest:
		n.onRequest(agent, bytes)
	case HdOk:
		n.onResponseOk(agent, bytes)
	case HdFail:
		n.onResponseFail(agent, bytes)
	case HdHeartbeat:
		n.onHeartbeat(agent, bytes)
	case HdNotify:
		n.onNotify(agent, bytes)
	case HdWatch:
		n.onWatchNotify(agent, bytes)
	default:
		kiwi.Error2(util.EcNotExist, util.M{
			"head": bytes[0],
		})
	}
}

func (n *node) onHeartbeat(agent kiwi.IAgent, bytes []byte) {

}

func (n *node) onPush(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvPusPkt()
	err := kiwi.Packer().UnpackPush(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnPush(pkt)
}

func (n *node) onRequest(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvReqPkt()
	err := kiwi.Packer().UnpackRequest(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnRequest(pkt)
}

func (n *node) onResponseOk(agent kiwi.IAgent, bytes []byte) {
	tid, payload, err := kiwi.Packer().UnpackResponseOk(bytes)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnResponseOkBytes(tid, payload)
}

func (n *node) onResponseFail(agent kiwi.IAgent, bytes []byte) {
	tid, code, err := kiwi.Packer().UnpackResponseFail(bytes)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.TE(tid, err)
		return
	}
	kiwi.Router().OnResponseFail(tid, code)
}

func (n *node) onNotify(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvNtfPkt()
	err := kiwi.Packer().UnpackNotify(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}

	for _, service := range AllService() {
		service.OnNotice(pkt)
	}
}

func (n *node) onWatchNotify(agent kiwi.IAgent, bytes []byte) {
	meta := util.M{}
	nodeId, codes, err := kiwi.Packer().UnpackWatchNotify(bytes, meta)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Node().ReceiveWatchNotice(nodeId, codes, meta)
}

type nodeJobConnect struct {
	meta kiwi.SvcMeta
}

type nodeJobConnected struct {
	dialer *nodeDialer
}

type nodeJobDisconnect struct {
	svc    kiwi.TSvc
	nodeId int64
}

type nodeJobDisconnected struct {
	dialer *nodeDialer
	err    *util.Err
}

type nodeJobSendNotice struct {
	notice kiwi.ISndNotice
	filter util.MToBool
}

type nodeJobSendNoticeOne struct {
	notice kiwi.ISndNotice
	filter util.MToBool
}

type nodeJobWatchNotice struct {
	nodeId int64
	codes  []kiwi.TCode
	meta   util.M
}

type nodeJobPus struct {
	pus kiwi.ISndPush
}

type nodeJobPusNode struct {
	nodeId int64
	pus    kiwi.ISndPush
}

type nodeJobReq struct {
	req kiwi.ISndRequest
}

type nodeJobReqNode struct {
	nodeId int64
	req    kiwi.ISndRequest
}

type nodeJobSendBytes struct {
	nodeId  int64
	payload []byte
	fnErr   util.FnErr
}
