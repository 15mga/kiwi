package kiwi

import (
	"github.com/15mga/kiwi/util"
	"reflect"
)

var (
	_Codec ICodec
)

func Codec() ICodec {
	return _Codec
}

func SetCodec(codec ICodec) {
	_Codec = codec
}

type ICodec interface {
	PbMarshal(obj util.IMsg) ([]byte, *util.Err)
	PbUnmarshal(data []byte, msg util.IMsg) *util.Err
	PbUnmarshal2(svc TSvc, mtd TCode, data []byte) (util.IMsg, *util.Err)
	JsonMarshal(obj util.IMsg) ([]byte, *util.Err)
	JsonUnmarshal(data []byte, msg util.IMsg) *util.Err
	JsonUnmarshal2(svc TSvc, mtd TCode, data []byte) (util.IMsg, *util.Err)
	Spawn(svc TSvc, mtd TCode) (util.IMsg, *util.Err)
	SpawnRes(svc TSvc, mtd TCode) (util.IMsg, *util.Err)
	BindFac(svc TSvc, mtd TCode, fac util.ToMsg)
	BindReqToRes(svc TSvc, req, res TCode)
	ReqToResCode(svc TSvc, req TCode) (TCode, *util.Err)
	MsgToSvcCode(msg util.IMsg) (svc TSvc, code TCode)
}

func CodecSpawn[T any](svc TSvc, mtd TCode) (T, *util.Err) {
	o, err := _Codec.Spawn(svc, mtd)
	if err != nil {
		return util.Default[T](), err
	}
	t, ok := o.(T)
	if !ok {
		return t, util.NewErr(util.EcWrongType, util.M{
			"service":  svc,
			"method":   mtd,
			"expected": reflect.TypeOf(t).Name(),
			"actual":   reflect.TypeOf(o).Name(),
		})
	}
	return t, nil
}

func CodecSpawnRes[T any](svc TSvc, mtd TCode) (T, *util.Err) {
	resMtd, err := _Codec.ReqToResCode(svc, mtd)
	if err != nil {
		return util.Default[T](), err
	}
	return CodecSpawn[T](svc, resMtd)
}
