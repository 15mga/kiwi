package kiwi

import "github.com/15mga/kiwi/util"

type ISndPacket interface {
	InitHead()
	Pid() int64
	Tid() int64
	Json() bool
	Svc() TSvc
	Code() TCode
	Head() util.M
	GetSvcNodeId() (int64, bool)
	Payload() []byte
	Msg() util.IMsg
	Dispose()
}

type ISndRequest interface {
	ISndPacket
	SetBytesHandler(fail util.FnUint16, ok util.FnBytes)
	SetHandler(fail util.FnUint16, ok util.FnMsg)
	SetChHandler(failCh chan<- uint16, okCh chan<- util.IMsg)
	SetBytesChHandler(failCh chan<- uint16, okCh chan<- []byte)
	OkBytes(bytes []byte)
	Ok(res util.IMsg)
	Fail(code uint16)
	Error(err *util.Err)
}

type ISndPush interface {
	ISndPacket
}

type ISndNotice interface {
	ISndPacket
}
