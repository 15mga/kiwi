package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync/atomic"
)

func NewRcvPusPkt() *RcvPusPkt {
	return &RcvPusPkt{}
}

func NewRcvReqPkt() *RcvReqPkt {
	return &RcvReqPkt{}
}

func NewRcvNtfPkt() *RcvNtcPkt {
	return &RcvNtcPkt{}
}

type rcvPkt struct {
	headId     string
	senderId   int64
	msgType    uint8
	tid        int64
	svc        kiwi.TSvc
	code       kiwi.TCode
	head       util.M
	json       bool
	msg        util.IMsg
	workerType kiwi.EWorker
	workerKey  string
	completed  int32
}

func (p *rcvPkt) Worker() kiwi.EWorker {
	return p.workerType
}

func (p *rcvPkt) WorkerKey() string {
	return p.workerKey
}

func (p *rcvPkt) SetWorker(typ kiwi.EWorker, key string) {
	p.workerType = typ
	p.workerKey = key
}

func (p *rcvPkt) InitWithBytes(msgType uint8, tid int64, head util.M, json bool, payload []byte) *util.Err {
	var (
		msg util.IMsg
		err *util.Err
	)
	p.svc, _ = util.MGet[kiwi.TSvc](head, HeadSvc)
	p.code, _ = util.MGet[kiwi.TCode](head, HeadCode)
	if json {
		msg, err = kiwi.Codec().JsonUnmarshal2(p.svc, p.code, payload)
	} else {
		msg, err = kiwi.Codec().PbUnmarshal2(p.svc, p.code, payload)
	}
	if err != nil {
		return err
	}
	p.msgType = msgType
	p.tid = tid
	p.head = head
	p.json = json
	p.msg = msg
	p.headId, _ = util.MGet[string](p.head, HeadId)
	p.senderId, _ = util.MGet[int64](p.head, HeadSndId)
	atomic.AddUint64(&_ReceivePktCount, 1)
	return nil
}

func (p *rcvPkt) InitWithMsg(msgType uint8, tid int64, head util.M, json bool, msg util.IMsg) {
	p.msgType = msgType
	p.tid = tid
	p.head = head
	p.json = json
	p.msg = msg
	p.headId, _ = util.MGet[string](p.head, HeadId)
	p.svc, _ = util.MGet[kiwi.TSvc](p.head, HeadSvc)
	p.code, _ = util.MGet[kiwi.TCode](p.head, HeadCode)
	p.senderId, _ = util.MGet[int64](p.head, HeadSndId)
	atomic.AddUint64(&_ReceivePktCount, 1)
}

func (p *rcvPkt) SenderId() int64 {
	return p.senderId
}

func (p *rcvPkt) Tid() int64 {
	return p.tid
}

func (p *rcvPkt) Svc() kiwi.TSvc {
	return p.svc
}

func (p *rcvPkt) Code() kiwi.TCode {
	return p.code
}

func (p *rcvPkt) Head() util.M {
	return p.head
}

func (p *rcvPkt) HeadId() string {
	return p.headId
}

func (p *rcvPkt) Json() bool {
	return p.json
}

func (p *rcvPkt) Msg() util.IMsg {
	return p.msg
}

func (p *rcvPkt) Complete() {
	if atomic.CompareAndSwapInt32(&p.completed, 0, 1) {
		atomic.AddUint64(&_CompletePktCount, 1)
	}
}

func (p *rcvPkt) Err(err *util.Err) {
	if err != nil {
		kiwi.TE(p.tid, err)
	}
	p.Complete()
}

func (p *rcvPkt) Err2(code util.TErrCode, m util.M) {
	kiwi.TE2(p.tid, code, m)
	p.Complete()
}

func (p *rcvPkt) Err3(code util.TErrCode, e error) {
	kiwi.TE3(p.tid, code, e)
	p.Complete()
}

var (
	_ReceivePktCount       uint64
	_CompletePktCount      uint64
	_ResponseSendFailCount uint64
)

func ReceivePktCount() uint64 {
	return atomic.LoadUint64(&_ReceivePktCount)
}

func CompletePktCount() uint64 {
	return atomic.LoadUint64(&_CompletePktCount)
}

func ResponseSendFailCount() uint64 {
	return atomic.LoadUint64(&_ResponseSendFailCount)
}
