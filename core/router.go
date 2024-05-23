package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"github.com/orcaman/concurrent-map/v2"
	"github.com/panjf2000/ants/v2"
)

func InitRouter() {
	s := &router{
		pusHandle: make(map[kiwi.TSvcCode]kiwi.FnRcvPus),
		reqHandle: make(map[kiwi.TSvcCode]kiwi.FnRcvReq),
		idToRequest: cmap.NewWithCustomShardingFunction[int64, kiwi.ISndRequest](func(key int64) uint32 {
			return uint32(key)
		}),
	}
	kiwi.SetRouter(s)
}

type router struct {
	leaseId     int64
	pusHandle   map[kiwi.TSvcCode]kiwi.FnRcvPus
	reqHandle   map[kiwi.TSvcCode]kiwi.FnRcvReq
	idToRequest cmap.ConcurrentMap[int64, kiwi.ISndRequest]
}

func (s *router) OnPush(pkt kiwi.IRcvPush) {
	fn, ok := s.pusHandle[kiwi.MergeSvcCode(pkt.Svc(), pkt.Code())]
	if !ok {
		kiwi.TE(pkt.Tid(), util.NewErr(util.EcNotExist, util.M{
			"service": pkt.Svc(),
			"code":    pkt.Code(),
		}))
		return
	}
	fn(pkt)
}

func (s *router) OnRequest(pkt kiwi.IRcvRequest) {
	fn, ok := s.reqHandle[kiwi.MergeSvcCode(pkt.Svc(), pkt.Code())]
	if !ok {
		kiwi.TE(pkt.Tid(), util.NewErr(util.EcNotExist, util.M{
			"service": pkt.Svc(),
			"code":    pkt.Code(),
		}))
		return
	}
	pkt.Head()["rcd"], _ = kiwi.Codec().ReqToResCode(pkt.Svc(), pkt.Code())
	fn(pkt)
}

func (s *router) BindPus(svc kiwi.TSvc, code kiwi.TCode, fn kiwi.FnRcvPus) {
	s.pusHandle[kiwi.MergeSvcCode(svc, code)] = fn
}

func (s *router) BindReq(svc kiwi.TSvc, code kiwi.TCode, fn kiwi.FnRcvReq) {
	s.reqHandle[kiwi.MergeSvcCode(svc, code)] = fn
}

func (s *router) AddRequest(req kiwi.ISndRequest) {
	s.idToRequest.Set(req.Tid(), req)
}

func (s *router) DelRequest(tid int64) {
	s.idToRequest.Remove(tid)
}

func (s *router) OnResponseOk(tid int64, head util.M, msg util.IMsg) {
	req, ok := s.idToRequest.Pop(tid)
	if !ok {
		return
	}
	req.Ok(head, msg)
}

func (s *router) OnResponseOkBytes(tid int64, head util.M, bytes []byte) {
	req, ok := s.idToRequest.Pop(tid)
	if !ok {
		return
	}
	req.OkBytes(head, bytes)
}

func (s *router) OnResponseFail(tid int64, head util.M, code uint16) {
	req, ok := s.idToRequest.Pop(tid)
	if !ok {
		return
	}
	req.Fail(head, code)
}

func ActivePrcPus[Pus util.IMsg](pkt kiwi.IRcvPush, key string, handler func(kiwi.IRcvPush, Pus)) {
	pkt.SetWorker(kiwi.EWorkerActive, key)
	worker.Active().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvPush)
		kiwi.TI(pkt.Tid(), "push", util.M{
			"pus":  pkt,
			"name": pkt.Msg().ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, pkt.(Pus))
	}, pkt)
}

func SharePrcPus[Pus util.IMsg](pkt kiwi.IRcvPush, key string, handler func(kiwi.IRcvPush, Pus)) {
	pkt.SetWorker(kiwi.EWorkerShare, key)
	worker.Share().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvPush)
		kiwi.TI(pkt.Tid(), "push", util.M{
			"pus":  pkt,
			"name": pkt.Msg().ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, pkt.(Pus))
	}, pkt)
}

func GoPrcPus[Pus util.IMsg](pkt kiwi.IRcvPush, handler func(kiwi.IRcvPush, Pus)) {
	pkt.SetWorker(kiwi.EWorkerGo, "")
	e := ants.Submit(func() {
		pus := pkt.Msg().(Pus)
		kiwi.TI(pkt.Tid(), "push", util.M{
			"pus":  pus,
			"name": pus.ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, pus)
	})
	if e != nil {
		kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
	}
}

func GlobalPrcPus[Pus util.IMsg](pkt kiwi.IRcvPush, handler func(kiwi.IRcvPush, Pus)) {
	pkt.SetWorker(kiwi.EWorkerGlobal, "")
	worker.Global().Push(func(a any) {
		pkt := a.(kiwi.IRcvPush)
		kiwi.TI(pkt.Tid(), "push", util.M{
			"pus":  pkt,
			"name": pkt.Msg().ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, pkt.(Pus))
	}, pkt)
}

func SelfPrcPus[Pus util.IMsg](pkt kiwi.IRcvPush, handler func(kiwi.IRcvPush, Pus)) {
	pkt.SetWorker(kiwi.EWorkerSelf, "")
	pus := pkt.Msg().(Pus)
	kiwi.TI(pkt.Tid(), "push", util.M{
		"pus":  pus,
		"name": pus.ProtoReflect().Descriptor().Name(),
	})
	handler(pkt, pus)
}

func ActivePrcReq[Req, Res util.IMsg](pkt kiwi.IRcvRequest, key string, handler func(kiwi.IRcvRequest, Req, Res)) {
	pkt.SetWorker(kiwi.EWorkerActive, key)
	worker.Active().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvRequest)
		res, err := kiwi.CodecSpawnRes[Res](pkt.Svc(), pkt.Code())
		if err != nil {
			pkt.Err(err)
			return
		}
		req := pkt.Msg().(Req)
		handler(pkt, req, res)
	}, pkt)
}

func SharePrcReq[Req, Res util.IMsg](pkt kiwi.IRcvRequest, key string, handler func(kiwi.IRcvRequest, Req, Res)) {
	pkt.SetWorker(kiwi.EWorkerShare, key)
	worker.Share().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvRequest)
		code := pkt.Code()
		res, err := kiwi.CodecSpawnRes[Res](pkt.Svc(), code)
		if err != nil {
			pkt.Err(err)
			return
		}
		req := pkt.Msg().(Req)
		handler(pkt, req, res)
	}, pkt)
}

func GoPrcReq[Req, Res util.IMsg](pkt kiwi.IRcvRequest, handler func(kiwi.IRcvRequest, Req, Res)) {
	pkt.SetWorker(kiwi.EWorkerGo, "")
	e := ants.Submit(func() {
		code := pkt.Code()
		res, err := kiwi.CodecSpawnRes[Res](pkt.Svc(), code)
		if err != nil {
			pkt.Err(err)
			return
		}
		req := pkt.Msg().(Req)
		handler(pkt, req, res)
	})
	if e != nil {
		kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
	}
}

func GlobalPrcReq[Req, Res util.IMsg](pkt kiwi.IRcvRequest, handler func(kiwi.IRcvRequest, Req, Res)) {
	pkt.SetWorker(kiwi.EWorkerGlobal, "")
	worker.Global().Push(func(a any) {
		pkt := a.(kiwi.IRcvRequest)
		code := pkt.Code()
		res, err := kiwi.CodecSpawnRes[Res](pkt.Svc(), code)
		if err != nil {
			pkt.Err(err)
			return
		}
		req := pkt.Msg().(Req)
		handler(pkt, req, res)
	}, pkt)
}

func SelfPrcReq[Req, Res util.IMsg](pkt kiwi.IRcvRequest, handler func(kiwi.IRcvRequest, Req, Res)) {
	pkt.SetWorker(kiwi.EWorkerSelf, "")
	code := pkt.Code()
	res, err := kiwi.CodecSpawnRes[Res](pkt.Svc(), code)
	if err != nil {
		pkt.Err(err)
		return
	}
	req := pkt.Msg().(Req)
	handler(pkt, req, res)
}

func ActivePrcNtc[Ntc util.IMsg](pkt kiwi.IRcvNotice, key string, handler func(kiwi.IRcvNotice, Ntc)) {
	pkt.SetWorker(kiwi.EWorkerActive, key)
	worker.Active().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvNotice)
		ntc := pkt.Msg().(Ntc)
		kiwi.TI(pkt.Tid(), "notice", util.M{
			"ntc":  ntc,
			"name": ntc.ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, ntc)
	}, pkt)
}

func SharePrcNtc[Ntc util.IMsg](pkt kiwi.IRcvNotice, key string, handler func(kiwi.IRcvNotice, Ntc)) {
	pkt.SetWorker(kiwi.EWorkerShare, key)
	worker.Share().Push(key, func(a any) {
		pkt := a.(kiwi.IRcvNotice)
		ntc := pkt.Msg().(Ntc)
		kiwi.TI(pkt.Tid(), "notice", util.M{
			"ntc":  ntc,
			"name": ntc.ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, ntc)
	}, pkt)
}

func GoPrcNtc[Ntc util.IMsg](pkt kiwi.IRcvNotice, handler func(kiwi.IRcvNotice, Ntc)) {
	pkt.SetWorker(kiwi.EWorkerGo, "")
	e := ants.Submit(func() {
		ntc := pkt.Msg().(Ntc)
		kiwi.TI(pkt.Tid(), "notice", util.M{
			"ntc":  ntc,
			"name": ntc.ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, ntc)
	})
	if e != nil {
		kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
	}
}

func GlobalPrcNtc[Ntc util.IMsg](pkt kiwi.IRcvNotice, handler func(kiwi.IRcvNotice, Ntc)) {
	pkt.SetWorker(kiwi.EWorkerGlobal, "")
	worker.Global().Push(func(a any) {
		pkt := a.(kiwi.IRcvNotice)
		ntc := pkt.Msg().(Ntc)
		kiwi.TI(pkt.Tid(), "notice", util.M{
			"ntc":  ntc,
			"name": ntc.ProtoReflect().Descriptor().Name(),
		})
		handler(pkt, ntc)
	}, pkt)
}

func SelfPrcNtc[Ntc util.IMsg](pkt kiwi.IRcvNotice, handler func(kiwi.IRcvNotice, Ntc)) {
	pkt.SetWorker(kiwi.EWorkerSelf, "")
	ntc := pkt.Msg().(Ntc)
	kiwi.TI(pkt.Tid(), "notice", util.M{
		"ntc":  ntc,
		"name": ntc.ProtoReflect().Descriptor().Name(),
	})
	handler(pkt, ntc)
}

var (
	_MsgNameToSvcCode = map[string]kiwi.TSvcCode{}
)

func BindMsgToSvcCode(msg util.IMsg, svc kiwi.TSvc, code kiwi.TCode) {
	fullName := msg.ProtoReflect().Descriptor().Name()
	_MsgNameToSvcCode[string(fullName)] = kiwi.MergeSvcCode(svc, code)
}

func MsgToSvcCode(msg util.IMsg) (svc kiwi.TSvc, code kiwi.TCode, ok bool) {
	fullName := msg.ProtoReflect().Descriptor().Name()
	sm, ok := _MsgNameToSvcCode[string(fullName)]
	if !ok {
		return
	}
	svc, code = kiwi.SplitSvcCode(sm)
	return
}
