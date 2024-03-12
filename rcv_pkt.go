package kiwi

import "github.com/15mga/kiwi/util"

type FnRcvPkt func(IRcvPkt)
type FnRcvPus func(IRcvPush)
type FnRcvReq func(IRcvRequest)

type EWorker uint8

const (
	EWorkerGo     EWorker = iota
	EWorkerActive         //需要key
	EWorkerShare          //需要key
	EWorkerGlobal
	EWorkerSelf
)

type IRcvPkt interface {
	SenderId() int64
	Tid() int64
	Svc() TSvc
	Code() TCode
	Head() util.M
	HeadId() string
	Json() bool
	Msg() util.IMsg
	SetWorker(typ EWorker, key string)
	Worker() EWorker
	WorkerKey() string
	InitWithBytes(msgType uint8, tid int64, head util.M, json bool, bytes []byte) *util.Err
	InitWithMsg(msgType uint8, tid int64, head util.M, json bool, msg util.IMsg)
	Complete()
	Err(err *util.Err)
	Err2(code util.TErrCode, m util.M)
	Err3(code util.TErrCode, e error)
}

type IRcvPush interface {
	IRcvPkt
}

type IRcvRequest interface {
	IRcvPkt
	Ok(msg util.IMsg)
	Fail(code uint16)
}

type IRcvNotice interface {
	IRcvPkt
}
