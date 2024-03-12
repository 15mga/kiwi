package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

func InitCodec() {
	kiwi.SetCodec(&codec{
		fac:          make(map[kiwi.TSvcCode]util.ToMsg),
		msgToSvcCode: make(map[string]kiwi.TSvcCode),
		reqToRes:     make(map[kiwi.TSvcCode]kiwi.TCode),
	})
}

type codec struct {
	fac          map[kiwi.TSvcCode]util.ToMsg
	msgToSvcCode map[string]kiwi.TSvcCode
	reqToRes     map[kiwi.TSvcCode]kiwi.TCode
}

func (c *codec) PbMarshal(obj util.IMsg) ([]byte, *util.Err) {
	if obj == nil {
		return nil, nil
	}
	bytes, e := util.PbMarshal(obj)
	if e != nil {
		return nil, util.WrapErr(util.EcMarshallErr, e)
	}
	return bytes, nil
}

func (c *codec) PbUnmarshal(data []byte, msg util.IMsg) *util.Err {
	if len(data) == 0 {
		return nil
	}
	return util.WrapErr(util.EcUnmarshallErr, util.PbUnmarshal(data, msg))
}

func (c *codec) PbUnmarshal2(svc kiwi.TSvc, code kiwi.TCode, data []byte) (util.IMsg, *util.Err) {
	fn, ok := c.fac[kiwi.MergeSvcCode(svc, code)]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"svc":  svc,
			"code": code,
		})
	}
	msg := fn()
	if len(data) == 0 {
		return msg, nil
	}
	e := util.PbUnmarshal(data, msg)
	if e != nil {
		return nil, util.WrapErr(util.EcUnmarshallErr, e)
	}
	return msg, nil
}

func (c *codec) JsonMarshal(msg util.IMsg) ([]byte, *util.Err) {
	if msg == nil {
		return nil, nil
	}
	bytes, e := util.JsonMarshal(msg)
	if e != nil {
		return nil, util.WrapErr(util.EcMarshallErr, e)
	}
	return bytes, nil
}

func (c *codec) JsonUnmarshal(data []byte, msg util.IMsg) *util.Err {
	if len(data) == 0 {
		return nil
	}
	return util.WrapErr(util.EcUnmarshallErr, util.JsonUnmarshal(data, msg))
}

func (c *codec) JsonUnmarshal2(svc kiwi.TSvc, code kiwi.TCode, data []byte) (util.IMsg, *util.Err) {
	fn, ok := c.fac[kiwi.MergeSvcCode(svc, code)]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"service": svc,
			"code":    code,
		})
	}
	msg := fn()
	if len(data) == 0 {
		return msg, nil
	}
	e := util.JsonUnmarshal(data, msg)
	if e != nil {
		e.AddParams(util.M{
			"service": svc,
			"code":    code,
			"data":    string(data),
		})
		return nil, util.WrapErr(util.EcUnmarshallErr, e)
	}
	return msg, nil
}

func (c *codec) Spawn(svc kiwi.TSvc, code kiwi.TCode) (util.IMsg, *util.Err) {
	fn, ok := c.fac[kiwi.MergeSvcCode(svc, code)]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"service": svc,
			"code":    code,
		})
	}
	return fn(), nil
}

func (c *codec) SpawnRes(svc kiwi.TSvc, code kiwi.TCode) (util.IMsg, *util.Err) {
	m, err := c.ReqToResCode(svc, code)
	if err != nil {
		return nil, err
	}
	return c.Spawn(svc, m)
}

func (c *codec) BindFac(svc kiwi.TSvc, code kiwi.TCode, fac util.ToMsg) {
	sm := kiwi.MergeSvcCode(svc, code)
	c.fac[sm] = fac
	msg := fac()
	msgName := string(msg.ProtoReflect().Descriptor().Name())
	c.msgToSvcCode[msgName] = sm
}

func (c *codec) BindReqToRes(svc kiwi.TSvc, req, res kiwi.TCode) {
	c.reqToRes[kiwi.MergeSvcCode(svc, req)] = res
}

func (c *codec) ReqToResCode(svc kiwi.TSvc, req kiwi.TCode) (kiwi.TCode, *util.Err) {
	res, ok := c.reqToRes[kiwi.MergeSvcCode(svc, req)]
	if !ok {
		return 0, util.NewErr(util.EcNotExist, util.M{
			"svc": svc,
			"req": req,
		})
	}
	return res, nil
}

func (c *codec) MsgToSvcCode(msg util.IMsg) (kiwi.TSvc, kiwi.TCode) {
	name := string(msg.ProtoReflect().Descriptor().Name())
	sc := c.msgToSvcCode[name]
	return kiwi.SplitSvcCode(sc)
}
