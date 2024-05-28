package core

import (
	"github.com/15mga/kiwi/worker"
	"github.com/panjf2000/ants/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

type ReqHandler int

const (
	ReqHandlerBytes ReqHandler = iota
	ReqHandlerMsg
	ReqHandlerCh
	ReqHandlerBytesCh
)

type SRequest struct {
	sndPkt
	handlerType ReqHandler
	timer       *time.Timer
	okBytes     util.FnBytes
	okMsg       util.FnMsg
	okCh        chan<- util.IMsg
	okBytesCh   chan<- []byte
	failCh      chan<- uint16
	fail        util.FnUint16
	disposed    int32
}

func (r *SRequest) SetBytesHandler(fail util.FnUint16, ok util.FnBytes) {
	r.handlerType = ReqHandlerBytes
	r.fail = fail
	r.okBytes = ok
}

func (r *SRequest) SetHandler(fail util.FnUint16, ok util.FnMsg) {
	r.handlerType = ReqHandlerMsg
	r.fail = fail
	r.okMsg = ok
}

func (r *SRequest) SetChHandler(failCh chan<- uint16, okCh chan<- util.IMsg) {
	r.handlerType = ReqHandlerCh
	r.failCh = failCh
	r.okCh = okCh
}

func (r *SRequest) SetBytesChHandler(failCh chan<- uint16, okCh chan<- []byte) {
	r.handlerType = ReqHandlerBytesCh
	r.failCh = failCh
	r.okBytesCh = okCh
}

func (r *SRequest) OkBytes(bytes []byte) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	switch r.handlerType {
	case ReqHandlerBytes:
		if r.okBytes == nil {
			return
		}
		r.okBytes(bytes)
	case ReqHandlerMsg:
		if r.okMsg == nil {
			return
		}
		res, err := kiwi.Codec().SpawnRes(r.svc, r.code)
		if r.json {
			err = kiwi.Codec().JsonUnmarshal(bytes, res)
		} else {
			err = kiwi.Codec().PbUnmarshal(bytes, res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		r.okMsg(res)
	case ReqHandlerCh:
		if r.okCh == nil {
			return
		}
		res, err := kiwi.Codec().SpawnRes(r.svc, r.code)
		if err != nil {
			kiwi.Fatal(err)
			return
		}
		if r.json {
			err = kiwi.Codec().JsonUnmarshal(bytes, res)
		} else {
			err = kiwi.Codec().PbUnmarshal(bytes, res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		r.okCh <- res
	case ReqHandlerBytesCh:
		if r.okBytesCh == nil {
			return
		}
		r.okBytesCh <- bytes
	}
}

func (r *SRequest) Ok(res util.IMsg) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	switch r.handlerType {
	case ReqHandlerBytes:
		var (
			bytes []byte
			err   *util.Err
		)
		if r.json {
			bytes, err = kiwi.Codec().JsonMarshal(res)
		} else {
			bytes, err = kiwi.Codec().PbMarshal(res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		if r.okBytes != nil {
			r.okBytes(bytes)
		}
	case ReqHandlerMsg:
		if r.okMsg != nil {
			r.okMsg(res)
		}
	case ReqHandlerCh:
		if r.okCh != nil {
			r.okCh <- res
		}
	case ReqHandlerBytesCh:
		var (
			bytes []byte
			err   *util.Err
		)
		if r.json {
			bytes, err = kiwi.Codec().JsonMarshal(res)
		} else {
			bytes, err = kiwi.Codec().PbMarshal(res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		if r.okBytesCh != nil {
			r.okBytesCh <- bytes
		}
	}
}

func (r *SRequest) Fail(code uint16) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	switch r.handlerType {
	case ReqHandlerBytes:
		fallthrough
	case ReqHandlerMsg:
		if r.fail != nil {
			r.fail(code)
		}
	case ReqHandlerCh:
		fallthrough
	case ReqHandlerBytesCh:
		if r.failCh != nil {
			r.failCh <- code
		}
	}
}

func (r *SRequest) Error(err *util.Err) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()
	kiwi.TE(r.tid, err)
	r.fail(err.Code())
}

func (r *SRequest) timeout() {
	kiwi.TE2(r.tid, util.EcTimeout, r.head.Copy())
}

func (r *SRequest) isDisposed() bool {
	return atomic.LoadInt32(&r.disposed) == 1
}

func (r *SRequest) Dispose() {
	if atomic.CompareAndSwapInt32(&r.disposed, 0, 1) {
		r.sndPkt.Dispose()
		r.timer.Stop()
		_ReqPool.Put(r)
	}
}

var (
	ResponseTimeoutDur = time.Duration(5000) * time.Millisecond
	_ReqPool           = sync.Pool{
		New: func() any {
			return &SRequest{}
		},
	}
)

func newBytesRequest(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) *SRequest {
	if head == nil {
		head = util.M{}
	}
	GenHead(head)

	req := _ReqPool.Get().(*SRequest)
	req.pid = pid
	req.svc, req.code = svc, code
	req.json = json
	req.head = head
	req.payload = payload
	req.InitHead()
	req.timer = time.AfterFunc(ResponseTimeoutDur, req.timeout)
	req.tid = kiwi.TC(pid, head, IsExcludeLog(svc, code))
	atomic.StoreInt32(&req.disposed, 0)
	return req
}

func newRequest(pid int64, head util.M, json bool, msg util.IMsg) *SRequest {
	var (
		payload []byte
		err     *util.Err
	)
	if json {
		payload, err = kiwi.Codec().JsonMarshal(msg)
	} else {
		payload, err = kiwi.Codec().PbMarshal(msg)
	}
	if err != nil {
		kiwi.Fatal(err)
		return nil
	}
	svc, code := kiwi.Codec().MsgToSvcCode(msg)
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.msg = msg
	return req
}

func Req[ResT util.IMsg](pid int64, head util.M, req util.IMsg) (res ResT, code uint16) {
	request := newRequest(pid, head, false, req)
	failCh := make(chan uint16, 1)
	okCh := make(chan util.IMsg, 1)
	request.SetChHandler(failCh, okCh)
	kiwi.Router().AddRequest(request)
	kiwi.Node().Request(request)
	select {
	case code = <-failCh:
	case r := <-okCh:
		res = r.(ResT)
	}
	return
}

func ReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) ([]byte, uint16) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	okCh := make(chan []byte)
	failCh := make(chan uint16, 1)
	req.SetBytesChHandler(failCh, okCh)
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	select {
	case res := <-okCh:
		return res, 0
	case code := <-failCh:
		return nil, code
	}
}

func ReqNode(nodeId, pid int64, head util.M, req util.IMsg) (res util.IMsg, code uint16) {
	request := newRequest(pid, head, false, req)
	failCh := make(chan uint16, 1)
	okCh := make(chan util.IMsg, 1)
	request.SetChHandler(failCh, okCh)
	kiwi.Router().AddRequest(request)
	kiwi.Node().RequestNode(nodeId, request)
	select {
	case code = <-failCh:
	case res = <-okCh:
	}
	return
}

func ReqNodeBytes(nodeId, pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) ([]byte, uint16) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	okCh := make(chan []byte, 1)
	failCh := make(chan uint16, 1)
	req.SetBytesHandler(func(code uint16) {
		failCh <- code
	}, func(payload []byte) {
		okCh <- payload
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
	select {
	case res := <-okCh:
		return res, 0
	case fail := <-failCh:
		return nil, fail
	}
}

func AsyncReq(pid int64, head util.M, req util.IMsg, onFail util.FnUint16, onOk func(util.IMsg)) int64 {
	request := newRequest(pid, head, false, req)
	request.SetHandler(onFail, onOk)
	kiwi.Router().AddRequest(request)
	kiwi.Node().Request(request)
	return request.tid
}

func AsyncReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnUint16, onOk util.FnBytes) int64 {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	return req.tid
}

func AsyncReqNode[ResT util.IMsg](pid, nodeId int64, head util.M, req util.IMsg, onFail util.FnUint16, onOk func(ResT)) int64 {
	request := newRequest(pid, head, false, req)
	request.SetHandler(onFail, func(msg util.IMsg) {
		if onOk != nil {
			onOk(msg.(ResT))
		}
	})
	kiwi.Router().AddRequest(request)
	kiwi.Node().RequestNode(nodeId, request)
	return request.tid
}

func AsyncReqNodeBytes(pid, nodeId int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnUint16, onOk util.FnBytes) int64 {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
	return req.tid
}

func AsyncReqNodeBytesWithHead(pid, nodeId int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnUint16, onOk util.FnBytes) int64 {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
	return req.tid
}

func AsyncSubReq(pkt kiwi.IRcvRequest, req util.IMsg, resFail util.FnUint16, resOk func(util.IMsg)) int64 {
	head := util.M{}
	pkt.Head().CopyTo(head)
	switch pkt.Worker() {
	case kiwi.EWorkerGo:
		return AsyncReq(pkt.Tid(), head, req, func(code uint16) {
			if resFail == nil {
				return
			}
			e := ants.Submit(func() {
				resFail(code)
			})
			if e != nil {
				kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
			}
		}, func(res util.IMsg) {
			if resOk == nil {
				return
			}
			e := ants.Submit(func() {
				resOk(res)
			})
			if e != nil {
				kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
			}
		})
	case kiwi.EWorkerActive:
		return AsyncReq(pkt.Tid(), head, req, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Active().Push(pkt.WorkerKey(), func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func(res util.IMsg) {
			if resOk == nil {
				return
			}
			worker.Active().Push(pkt.WorkerKey(), func(data any) {
				resOk(data.(util.IMsg))
			}, res)
		})
	case kiwi.EWorkerShare:
		return AsyncReq(pkt.Tid(), head, req, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Share().Push(pkt.WorkerKey(), func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func(res util.IMsg) {
			if resOk == nil {
				return
			}
			worker.Share().Push(pkt.WorkerKey(), func(data any) {
				resOk(data.(util.IMsg))
			}, res)
		})
	case kiwi.EWorkerGlobal:
		return AsyncReq(pkt.Tid(), head, req, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Global().Push(func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func(res util.IMsg) {
			if resOk == nil {
				return
			}
			worker.Global().Push(func(data any) {
				resOk(data.(util.IMsg))
			}, res)
		})
	case kiwi.EWorkerSelf:
		return AsyncReq(pkt.Tid(), head, req, resFail, resOk)
	default:
		return 0
	}
}
