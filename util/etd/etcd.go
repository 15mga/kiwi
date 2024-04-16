package etd

import (
	"github.com/15mga/kiwi/util"
	etcd "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	_Etcd    *etcd.Client
	_LockTtl = 10
)

func Client() *etcd.Client {
	return _Etcd
}

func Conn(cfg etcd.Config) *util.Err {
	client, e := etcd.New(cfg)
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	_Etcd = client
	return nil
}

func Grant(ttl int64) (int64, *util.Err) {
	res, e := _Etcd.Grant(util.Ctx(), ttl)
	if e != nil {
		return 0, util.WrapErr(util.EcDiscoveryErr, e)
	}
	id := res.ID
	ch, e := _Etcd.KeepAlive(util.Ctx(), id)
	if e != nil {
		return 0, util.WrapErr(util.EcDiscoveryErr, e)
	}
	go func() {
		for r := range ch {
			if r != nil {
				continue
			}
			return
		}
	}()
	return int64(id), nil
}

func Revoke(id int64) *util.Err {
	_, e := _Etcd.Revoke(util.Ctx(), etcd.LeaseID(id))
	return util.WrapErr(util.EcDiscoveryErr, e)
}

func Del(key string, opts ...etcd.OpOption) *util.Err {
	_, e := _Etcd.Delete(util.Ctx(), key, opts...)
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	return nil
}

func Put(key, val string) *util.Err {
	_, e := _Etcd.Put(util.Ctx(), key, val)
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	return nil
}

func PutWithTtl(key, val string, ttl int64) (int64, *util.Err) {
	if ttl <= 0 {
		return 0, util.NewErr(util.EcParamsErr, util.M{
			"ttl": ttl,
		})
	}
	id, err := Grant(ttl)
	if err != nil {
		return 0, err
	}
	_, e := _Etcd.Put(util.Ctx(), key, val, etcd.WithLease(etcd.LeaseID(id)))
	return id, util.WrapErr(util.EcDiscoveryErr, e)
}

func Lock(key string, fn util.ToErr) *util.Err {
	if fn == nil {
		return nil
	}

	s, e := concurrency.NewSession(_Etcd, concurrency.WithTTL(_LockTtl))
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	defer s.Close()
	m := concurrency.NewMutex(s, key)

	e = m.Lock(util.Ctx())
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	err := fn()
	_ = m.Unlock(util.Ctx())
	return err
}

func TryLock(key string, fn util.Fn) *util.Err {
	if fn == nil {
		return nil
	}

	s, e := concurrency.NewSession(_Etcd, concurrency.WithTTL(_LockTtl))
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	defer s.Close()
	m := concurrency.NewMutex(s, key)

	e = m.TryLock(util.Ctx())
	if e != nil {
		return util.WrapErr(util.EcDiscoveryErr, e)
	}
	fn()
	_ = m.Unlock(util.Ctx())
	return nil
}

func Get(key string, fn util.StrBytesToBool, opts ...etcd.OpOption) (bool, *util.Err) {
	res, e := _Etcd.Get(util.Ctx(), key, opts...)
	if e != nil {
		return false, util.NewErr(util.EcDiscoveryErr, util.M{
			"key": key,
		})
	}
	if len(res.Kvs) == 0 {
		return false, nil
	}
	for _, kv := range res.Kvs {
		if ok := fn(string(kv.Key), kv.Value); !ok {
			return false, nil
		}
	}
	return true, nil
}

func GetOne(key string, opts ...etcd.OpOption) ([]byte, *util.Err) {
	res, e := _Etcd.Get(util.Ctx(), key, opts...)
	if e != nil {
		return nil, util.NewErr(util.EcDiscoveryErr, util.M{
			"key": key,
		})
	}
	if len(res.Kvs) == 0 {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"key": key,
		})
	}
	return res.Kvs[0].Value, nil
}

func Has(key string, opts ...etcd.OpOption) (bool, *util.Err) {
	exist := false
	_, err := Get(key, func(s string, bytes []byte) bool {
		exist = true
		return true
	}, opts...)
	if err != nil {
		return false, err
	}
	return exist, nil
}
