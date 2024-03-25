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
	SetBytesHandler(fail util.FnInt64MUint16, ok util.FnInt64MBytes)
	SetHandler(fail util.FnInt64MUint16, ok util.FnInt64MMsg)
	OkBytes(head util.M, bytes []byte)
	Ok(head util.M, msg util.IMsg)
	Fail(head util.M, code uint16)
	Error(err *util.Err)
}

type ISndPush interface {
	ISndPacket
}

type ISndNotice interface {
	ISndPacket
}
