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
)

type SRequest struct {
	sndPkt
	handlerType ReqHandler
	timer       *time.Timer
	okBytes     util.FnBytes
	okMsg       util.Fn
	okCh        chan<- uint16
	fail        util.FnUint16
	res         util.IMsg
	disposed    int32
}

func (r *SRequest) SetBytesHandler(fail util.FnUint16, ok util.FnBytes) {
	r.handlerType = ReqHandlerBytes
	r.okBytes = ok
	r.fail = fail
}

func (r *SRequest) SetHandler(res util.IMsg, fail util.FnUint16, ok util.Fn) {
	r.handlerType = ReqHandlerMsg
	r.okMsg = ok
	r.fail = fail
	r.res = res
}

func (r *SRequest) SetChHandler(res util.IMsg, ch chan<- uint16) {
	r.handlerType = ReqHandlerCh
	r.okCh = ch
	r.res = res
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
		var err *util.Err
		if r.json {
			err = kiwi.Codec().JsonUnmarshal(bytes, r.res)
		} else {
			err = kiwi.Codec().PbUnmarshal(bytes, r.res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		r.okMsg()
	case ReqHandlerCh:
		if r.okCh == nil {
			return
		}
		var err *util.Err
		if r.json {
			err = kiwi.Codec().JsonUnmarshal(bytes, r.res)
		} else {
			err = kiwi.Codec().PbUnmarshal(bytes, r.res)
		}
		if err != nil {
			r.Error(err)
			return
		}
		close(r.okCh)
	}
}

func (r *SRequest) Ok(res util.IMsg) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	r.res = res
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
			r.okMsg()
		}
	case ReqHandlerCh:
		if r.okCh != nil {
			close(r.okCh)
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
		if r.fail != nil {
			r.fail(code)
		}
	case ReqHandlerMsg:
		if r.fail != nil {
			r.fail(code)
		}
	case ReqHandlerCh:
		if r.okCh != nil {
			r.okCh <- code
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

func Req(pid int64, head util.M, req, res util.IMsg) uint16 {
	request := newRequest(pid, head, false, req)
	ch := make(chan uint16, 1)
	request.SetChHandler(res, ch)
	kiwi.Router().AddRequest(request)
	kiwi.Node().Request(request)
	return <-ch
}

func ReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) ([]byte, uint16) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	okCh := make(chan []byte)
	failCh := make(chan uint16, 1)
	req.SetBytesHandler(func(code uint16) {
		failCh <- code
	}, func(payload []byte) {
		okCh <- payload
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	select {
	case res := <-okCh:
		return res, 0
	case code := <-failCh:
		return nil, code
	}
}

func ReqNode(nodeId, pid int64, head util.M, req, res util.IMsg) uint16 {
	request := newRequest(pid, head, false, req)
	ch := make(chan uint16, 1)
	request.SetChHandler(res, ch)
	kiwi.Router().AddRequest(request)
	kiwi.Node().RequestNode(nodeId, request)
	return <-ch
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

func AsyncReq(pid int64, head util.M, req, res util.IMsg, onFail util.FnUint16, onOk util.Fn) {
	request := newRequest(pid, head, false, req)
	request.SetHandler(res, onFail, onOk)
	kiwi.Router().AddRequest(request)
	kiwi.Node().Request(request)
}

func AsyncReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnUint16, onOk util.FnBytes) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
}

func AsyncReqNode(pid, nodeId int64, head util.M, req, res util.IMsg, onFail util.FnUint16, onOk util.Fn) {
	request := newRequest(pid, head, false, req)
	request.SetHandler(res, onFail, onOk)
	kiwi.Router().AddRequest(request)
	kiwi.Node().RequestNode(nodeId, request)
}

func AsyncReqNodeBytes(pid, nodeId int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnUint16, onOk util.FnBytes) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
}

func AsyncSubReq(pkt kiwi.IRcvRequest, req, res util.IMsg, resFail util.FnUint16, resOk util.Fn) {
	head := util.M{}
	pkt.Head().CopyTo(head)
	switch pkt.Worker() {
	case kiwi.EWorkerGo:
		AsyncReq(pkt.Tid(), head, req, res, func(code uint16) {
			if resFail == nil {
				return
			}
			e := ants.Submit(func() {
				resFail(code)
			})
			if e != nil {
				kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
			}
		}, func() {
			if resOk == nil {
				return
			}
			e := ants.Submit(resOk)
			if e != nil {
				kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
			}
		})
	case kiwi.EWorkerActive:
		AsyncReq(pkt.Tid(), head, req, res, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Active().Push(pkt.WorkerKey(), func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func() {
			if resOk == nil {
				return
			}
			worker.Active().Push(pkt.WorkerKey(), func(data any) {
				resOk()
			}, nil)
		})
	case kiwi.EWorkerShare:
		AsyncReq(pkt.Tid(), head, req, res, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Share().Push(pkt.WorkerKey(), func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func() {
			if resOk == nil {
				return
			}
			worker.Share().Push(pkt.WorkerKey(), func(data any) {
				resOk()
			}, nil)
		})
	case kiwi.EWorkerGlobal:
		AsyncReq(pkt.Tid(), head, req, res, func(code uint16) {
			if resFail == nil {
				return
			}
			worker.Global().Push(func(data any) {
				resFail(data.(uint16))
			}, code)
		}, func() {
			if resOk == nil {
				return
			}
			worker.Global().Push(func(data any) {
				resOk()
			}, nil)
		})
	case kiwi.EWorkerSelf:
		AsyncReq(pkt.Tid(), head, req, res, func(code uint16) {
			if resFail == nil {
				return
			}
			resFail(code)
		}, resOk.Invoke)
	}
}
