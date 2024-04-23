package kiwi

import "github.com/15mga/kiwi/util"

var (
	_Packer IPacker
)

func Packer() IPacker {
	return _Packer
}

func SetPacker(packer IPacker) {
	_Packer = packer
}

type IPacker interface {
	PackWatchNotify(id int64, methods []TCode, meta util.M) []byte
	UnpackWatchNotify(bytes []byte, meta util.M) (id int64, methods []TCode, err *util.Err)
	PackPush(tid int64, pus ISndPush) ([]byte, *util.Err)
	UnpackPush(bytes []byte, pkg IRcvPush) (err *util.Err)
	UnpackPushBytes(bytes []byte, head util.M) (tid int64, json bool, payload []byte, err *util.Err)
	PackRequest(tid int64, req ISndRequest) ([]byte, *util.Err)
	UnpackRequest(bytes []byte, pkg IRcvRequest) (err *util.Err)
	PackResponseOk(tid int64, head util.M, pkt []byte) ([]byte, *util.Err)
	UnpackResponseOk(bytes []byte, head util.M) (tid int64, payload []byte, err *util.Err)
	PackResponseFail(tid int64, head util.M, code uint16) ([]byte, *util.Err)
	UnpackResponseFail(bytes []byte, head util.M) (tid int64, code uint16, readErr *util.Err)
	PackNotify(tid int64, ntf ISndNotice) ([]byte, *util.Err)
	UnpackNotify(bytes []byte, pkg IRcvNotice) (err *util.Err)
	PackM(m util.M) ([]byte, *util.Err)
	UnpackM(bytes []byte, m util.M) (err *util.Err)
}
