package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync/atomic"
	"time"
)

type RcvReqPkt struct {
	rcvPkt
}

func (p *RcvReqPkt) Ok(msg util.IMsg) {
	if !IsExcludeLog(p.svc, p.code) {
		sndTs, _ := util.MGet[int64](p.head, HeadSndTs)
		kiwi.TI(p.tid, "ok", util.M{
			"dur":  time.Now().UnixMilli() - sndTs,
			"name": p.msg.ProtoReflect().Descriptor().Name(),
			"req":  p.msg,
			"res":  msg,
		})
	}
	p.Complete()
	if p.senderId == kiwi.GetNodeMeta().NodeId {
		kiwi.Router().OnResponseOk(p.tid, p.head, msg)
		return
	}
	var (
		payload []byte
		err     *util.Err
	)
	if p.json {
		payload, err = kiwi.Codec().JsonMarshal(msg)
	} else {
		payload, err = kiwi.Codec().PbMarshal(msg)
	}
	if err != nil {
		kiwi.Error(err)
		return
	}
	res, err := kiwi.Packer().PackResponseOk(p.tid, p.head, payload)
	if err != nil {
		kiwi.Error(err)
		return
	}
	kiwi.Node().SendToNode(p.senderId, res, p.onSendErr)
}

func (p *RcvReqPkt) Err(err *util.Err) {
	if err.Code() < util.EcMin {
		p.Fail(err.Code())
		return
	}
	sndTs, _ := util.MGet[int64](p.head, HeadSndTs)
	err.AddParam("dur", time.Now().UnixMilli()-sndTs)
	err.AddParam("name", p.msg.ProtoReflect().Descriptor().Name())
	err.AddParam("req", p.msg)
	p.rcvPkt.Err(err)
	if p.senderId == kiwi.GetNodeMeta().NodeId {
		kiwi.Router().OnResponseFail(p.tid, p.head, err.Code())
		return
	}
	payload, e := kiwi.Packer().PackResponseFail(p.tid, p.head, err.Code())
	if e != nil {
		kiwi.Error(e)
		return
	}
	kiwi.Node().SendToNode(p.senderId, payload, p.onSendErr)
}

func (p *RcvReqPkt) Fail(code uint16) {
	if !IsExcludeLog(p.svc, p.code) {
		sndTs, _ := util.MGet[int64](p.head, HeadSndTs)
		kiwi.TI(p.tid, "fail", util.M{
			"dur":   time.Now().UnixMilli() - sndTs,
			"name":  p.msg.ProtoReflect().Descriptor().Name(),
			"req":   p.msg,
			"error": util.ErrCodeToStr(code),
		})
	}
	p.Complete()
	if p.senderId == kiwi.GetNodeMeta().NodeId {
		kiwi.Router().OnResponseFail(p.tid, p.head, code)
		return
	}
	payload, e := kiwi.Packer().PackResponseFail(p.tid, p.head, code)
	if e != nil {
		kiwi.Error(e)
		return
	}
	kiwi.Node().SendToNode(p.senderId, payload, p.onSendErr)
}

func (p *RcvReqPkt) onSendErr(err *util.Err) {
	if err == nil {
		return
	}
	atomic.AddUint64(&_ResponseSendFailCount, 1)
	kiwi.TE(p.tid, err)
}
