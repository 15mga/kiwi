package kiwi

import "github.com/15mga/kiwi/util"

var (
	_Router IRouter
)

func Router() IRouter {
	return _Router
}

func SetRouter(router IRouter) {
	_Router = router
}

type PktToKey func(pkt IRcvPkt) string

type WorkerFn func(id string, fn util.FnAnySlc, params ...any)

type IRouter interface {
	AddRequest(req ISndRequest)
	DelRequest(tid int64)
	BindPus(svc TSvc, code TCode, fn FnRcvPus)
	BindReq(svc TSvc, code TCode, fn FnRcvReq)
	OnPush(pkt IRcvPush)
	OnRequest(pkt IRcvRequest)
	OnResponseOk(tid int64, head util.M, msg util.IMsg)
	OnResponseOkBytes(tid int64, head util.M, bytes []byte)
	OnResponseFail(tid int64, head util.M, code uint16)
}
