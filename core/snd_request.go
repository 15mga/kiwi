package core

import (
	"github.com/15mga/kiwi/worker"
	"github.com/panjf2000/ants/v2"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

type SRequest struct {
	sndPkt
	isBytes  bool
	timer    *time.Timer
	okBytes  util.FnInt64MBytes
	okMsg    util.FnInt64MMsg
	fail     util.FnInt64MUint16
	disposed int32
}

func (r *SRequest) SetBytesHandler(fail util.FnInt64MUint16, ok util.FnInt64MBytes) {
	r.isBytes = true
	r.okBytes = ok
	r.fail = fail
}

func (r *SRequest) SetHandler(fail util.FnInt64MUint16, ok util.FnInt64MMsg) {
	r.isBytes = false
	r.okMsg = ok
	r.fail = fail
}

func (r *SRequest) OkBytes(head util.M, bytes []byte) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	if r.isBytes {
		if r.okBytes == nil {
			return
		}
		r.okBytes(r.tid, head, bytes)
	} else {
		if r.okMsg == nil {
			return
		}
		code, err := kiwi.Codec().ReqToResCode(r.svc, r.code)
		if err != nil {
			r.Error(err)
			return
		}
		var msg util.IMsg
		if r.json {
			msg, err = kiwi.Codec().JsonUnmarshal2(r.svc, code, bytes)
		} else {
			msg, err = kiwi.Codec().PbUnmarshal2(r.svc, code, bytes)
		}
		if err != nil {
			r.Error(err)
			return
		}
		r.okMsg(r.tid, head, msg)
	}
}

func (r *SRequest) Ok(head util.M, msg util.IMsg) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()

	if r.isBytes {
		var (
			bytes []byte
			err   *util.Err
		)
		if r.json {
			bytes, err = kiwi.Codec().JsonMarshal(msg)
		} else {
			bytes, err = kiwi.Codec().PbMarshal(msg)
		}
		if err != nil {
			r.Error(err)
			return
		}
		if r.okBytes != nil {
			r.okBytes(r.tid, head, bytes)
		}
	} else {
		if r.okMsg != nil {
			r.okMsg(r.tid, head, msg)
		}
	}
}

func (r *SRequest) Fail(head util.M, code uint16) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()
	r.fail(r.tid, head, code)
}

func (r *SRequest) Error(err *util.Err) {
	if r.isDisposed() {
		return
	}
	defer r.Dispose()
	kiwi.TE(r.tid, err)
	r.fail(r.tid, nil, err.Code())
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

func Req[ResT util.IMsg](pid int64, head util.M, msg util.IMsg) (ResT, util.M, uint16) {
	req := newRequest(pid, head, false, msg)
	okCh := make(chan util.IMsg, 1)
	failCh := make(chan uint16, 1)
	var rm util.M
	req.SetHandler(func(_ int64, m util.M, code uint16) {
		rm = m
		failCh <- code
	}, func(_ int64, m util.M, a util.IMsg) {
		rm = m
		okCh <- a
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	select {
	case res := <-okCh:
		r, ok := res.(ResT)
		if !ok {
			return util.Default[ResT](), nil, util.EcWrongType
		}
		return r, rm, 0
	case code := <-failCh:
		return util.Default[ResT](), rm, code
	}
}

func Req2(pid int64, head util.M, msg util.IMsg) (util.IMsg, util.M, uint16) {
	req := newRequest(pid, head, false, msg)
	okCh := make(chan util.IMsg)
	failCh := make(chan uint16, 1)
	var rm util.M
	req.SetHandler(func(_ int64, m util.M, code uint16) {
		rm = m
		failCh <- code
	}, func(_ int64, m util.M, a util.IMsg) {
		rm = m
		okCh <- a
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	select {
	case res := <-okCh:
		return res, rm, 0
	case code := <-failCh:
		return nil, rm, code
	}
}

func ReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) ([]byte, util.M, uint16) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	okCh := make(chan []byte)
	failCh := make(chan uint16, 1)
	var rm util.M
	req.SetBytesHandler(func(_ int64, m util.M, code uint16) {
		rm = m
		failCh <- code
	}, func(_ int64, m util.M, payload []byte) {
		rm = m
		okCh <- payload
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
	select {
	case res := <-okCh:
		return res, rm, 0
	case code := <-failCh:
		return nil, rm, code
	}
}

func ReqNode[ResT any](nodeId, pid int64, head util.M, msg util.IMsg) (ResT, util.M, uint16) {
	req := newRequest(pid, head, false, msg)
	okCh := make(chan util.IMsg, 1)
	failCh := make(chan uint16, 1)
	var rm util.M
	req.SetHandler(func(_ int64, m util.M, code uint16) {
		rm = m
		failCh <- code
	}, func(_ int64, m util.M, a util.IMsg) {
		rm = m
		okCh <- a
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
	select {
	case res := <-okCh:
		r, ok := res.(ResT)
		if !ok {
			return util.Default[ResT](), nil, util.EcWrongType
		}
		return r, rm, 0
	case code := <-failCh:
		return util.Default[ResT](), nil, code
	}
}

func ReqNodeBytes(nodeId, pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte) ([]byte, util.M, uint16) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	okCh := make(chan []byte, 1)
	failCh := make(chan uint16, 1)
	var rm util.M
	req.SetBytesHandler(func(_ int64, m util.M, code uint16) {
		rm = m
		failCh <- code
	}, func(_ int64, m util.M, payload []byte) {
		rm = m
		okCh <- payload
	})
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
	select {
	case res := <-okCh:
		return res, rm, 0
	case code := <-failCh:
		return nil, nil, code
	}
}

func AsyncReq(pid int64, head util.M, msg util.IMsg, onFail util.FnInt64MUint16, onOk util.FnInt64MMsg) {
	req := newRequest(pid, head, false, msg)
	req.SetHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
}

func AsyncReqBytes(pid int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnInt64MUint16, onOk util.FnInt64MBytes) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().Request(req)
}

func AsyncReqNode(pid, nodeId int64, head util.M, msg util.IMsg, onFail util.FnInt64MUint16, onOk util.FnInt64MMsg) {
	req := newRequest(pid, head, false, msg)
	req.SetHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
}

func AsyncReqNodeBytes(pid, nodeId int64, svc kiwi.TSvc, code kiwi.TCode, head util.M, json bool, payload []byte,
	onFail util.FnInt64MUint16, onOk util.FnInt64MBytes) {
	req := newBytesRequest(pid, svc, code, head, json, payload)
	req.SetBytesHandler(onFail, onOk)
	kiwi.Router().AddRequest(req)
	kiwi.Node().RequestNode(nodeId, req)
}

func AsyncSubReq[ResT util.IMsg](pkt kiwi.IRcvRequest, req util.IMsg, resFail util.FnInt64MUint16, resOk func(int64, util.M, ResT)) {
	head := util.M{}
	pkt.Head().CopyTo(head)
	switch pkt.Worker() {
	case kiwi.EWorkerGo:
		AsyncReq(pkt.Tid(), head, req, func(tid int64, head util.M, code uint16) {
			if resFail == nil {
				return
			}
			e := ants.Submit(func() {
				resFail(tid, head, code)
			})
			if e != nil {
				kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
			}
		}, func(tid int64, head util.M, msg util.IMsg) {
			if resOk == nil {
				return
			}
			res, ok := msg.(ResT)
			if ok {
				e := ants.Submit(func() {
					resOk(tid, head, res)
				})
				if e != nil {
					kiwi.TE3(pkt.Tid(), util.EcServiceErr, e)
				}
			} else {
				kiwi.TE2(pkt.Tid(), util.EcWrongType, util.M{
					"expected": reflect.TypeOf(util.Default[ResT]()),
					"actual":   reflect.TypeOf(msg),
				})
			}
		})
	case kiwi.EWorkerActive:
		AsyncReq(pkt.Tid(), head, req, func(tid int64, head util.M, code uint16) {
			if resFail == nil {
				return
			}
			worker.Active().Push(pkt.WorkerKey(), func(data any) {
				d := data.(sndReqJobActiveFail)
				resFail(d.tid, d.m, d.code)
			}, sndReqJobActiveFail{tid, head, code})
		}, func(tid int64, head util.M, msg util.IMsg) {
			if resOk == nil {
				return
			}
			res, ok := msg.(ResT)
			if ok {
				worker.Active().Push(pkt.WorkerKey(), func(data any) {
					d := data.(sndReqJobActiveOk)
					resOk(d.tid, d.m, d.res.(ResT))
				}, sndReqJobActiveOk{tid, head, res})
			} else {
				kiwi.TE2(pkt.Tid(), util.EcWrongType, util.M{
					"expected": reflect.TypeOf(util.Default[ResT]()),
					"actual":   reflect.TypeOf(msg),
				})
			}
		})
	case kiwi.EWorkerShare:
		AsyncReq(pkt.Tid(), head, req, func(tid int64, head util.M, code uint16) {
			if resFail == nil {
				return
			}
			worker.Share().Push(pkt.WorkerKey(), func(data any) {
				d := data.(sndReqJobActiveFail)
				resFail(d.tid, d.m, d.code)
			}, sndReqJobActiveFail{tid, head, code})
		}, func(tid int64, head util.M, msg util.IMsg) {
			if resOk == nil {
				return
			}
			res, ok := msg.(ResT)
			if ok {
				worker.Share().Push(pkt.WorkerKey(), func(data any) {
					d := data.(sndReqJobActiveOk)
					resOk(d.tid, d.m, d.res.(ResT))
				}, sndReqJobActiveOk{tid, head, res})
			} else {
				kiwi.TE2(pkt.Tid(), util.EcWrongType, util.M{
					"expected": reflect.TypeOf(util.Default[ResT]()),
					"actual":   reflect.TypeOf(msg),
				})
			}
		})
	case kiwi.EWorkerGlobal:
		AsyncReq(pkt.Tid(), head, req, func(tid int64, head util.M, code uint16) {
			if resFail == nil {
				return
			}
			worker.Global().Push(func(data any) {
				d := data.(sndReqJobActiveFail)
				resFail(d.tid, d.m, d.code)
			}, sndReqJobActiveFail{tid, head, code})
		}, func(tid int64, head util.M, msg util.IMsg) {
			if resOk == nil {
				return
			}
			res, ok := msg.(ResT)
			if ok {
				worker.Global().Push(func(data any) {
					d := data.(sndReqJobActiveOk)
					resOk(d.tid, d.m, d.res.(ResT))
				}, sndReqJobActiveOk{tid, head, res})
			} else {
				kiwi.TE2(pkt.Tid(), util.EcWrongType, util.M{
					"expected": reflect.TypeOf(util.Default[ResT]()),
					"actual":   reflect.TypeOf(msg),
				})
			}
		})
	case kiwi.EWorkerSelf:
		AsyncReq(pkt.Tid(), head, req, func(tid int64, head util.M, code uint16) {
			if resFail == nil {
				return
			}
			resFail(tid, head, code)
		}, func(tid int64, head util.M, msg util.IMsg) {
			if resOk == nil {
				return
			}
			res, ok := msg.(ResT)
			if ok {
				resOk(tid, head, res)
			} else {
				kiwi.TE2(pkt.Tid(), util.EcWrongType, util.M{
					"expected": reflect.TypeOf(util.Default[ResT]()),
					"actual":   reflect.TypeOf(msg),
				})
			}
		})
	}
}

type sndReqJobActiveFail struct {
	tid  int64
	m    util.M
	code uint16
}

type sndReqJobActiveOk struct {
	tid int64
	m   util.M
	res any
}
