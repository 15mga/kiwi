package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

func InitPacker() {
	kiwi.SetPacker(&packer{})
}

type packer struct {
}

func (p *packer) PackWatchNotify(id int64, codes []kiwi.TCode, meta util.M) []byte {
	var buffer util.ByteBuffer
	buffer.InitCap(11 + len(codes)*2)
	buffer.WUint8(HdWatch)
	buffer.WInt64(id)
	buffer.WUint16s(codes)
	_ = buffer.WMAny(meta)
	return buffer.All()
}

func (p *packer) UnpackWatchNotify(bytes []byte, meta util.M) (id int64, codes []kiwi.TCode, err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	id, err = buffer.RInt64()
	if err != nil {
		return
	}
	codes, err = buffer.RUint16s()
	if err != nil {
		return
	}
	err = buffer.RMAny(meta)
	return
}

func (p *packer) PackPush(tid int64, pus kiwi.ISndPush) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(256)
	buffer.WUint8(HdPush)
	buffer.WInt64(tid)
	err := buffer.WMAny(pus.Head())
	if err != nil {
		return nil, err
	}
	buffer.WBool(pus.Json())
	_, e := buffer.Write(pus.Payload())
	if e != nil {
		return nil, util.NewErr(util.EcWriteFail, util.M{
			"error": e,
		})
	}
	return buffer.All(), nil
}

func (p *packer) UnpackPush(bytes []byte, pkg kiwi.IRcvPush) (err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err := buffer.RInt64()
	if err != nil {
		return
	}
	head := make(util.M)
	err = buffer.RMAny(head)
	if err != nil {
		return
	}
	json, err := buffer.RBool()
	if err != nil {
		return
	}
	bs := buffer.RAvailable()
	return pkg.InitWithBytes(HdPush, tid, head, json, bs)
}

func (p *packer) UnpackPushBytes(bytes []byte, head util.M) (tid int64, json bool, payload []byte, err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err = buffer.RInt64()
	if err != nil {
		return
	}
	err = buffer.RMAny(head)
	if err != nil {
		return
	}
	json, err = buffer.RBool()
	if err != nil {
		return
	}
	payload = buffer.RAvailable()
	return
}

func (p *packer) PackRequest(tid int64, req kiwi.ISndRequest) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(256)
	buffer.WUint8(HdRequest)
	buffer.WInt64(kiwi.GetNodeMeta().NodeId)
	buffer.WInt64(tid)
	err := buffer.WMAny(req.Head())
	if err != nil {
		return nil, err
	}
	buffer.WBool(req.Json())
	_, e := buffer.Write(req.Payload())
	if e != nil {
		return nil, util.NewErr(util.EcWriteFail, util.M{
			"error": e,
		})
	}
	return buffer.All(), nil
}

func (p *packer) UnpackRequest(bytes []byte, pkg kiwi.IRcvRequest) (err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err := buffer.RInt64()
	if err != nil {
		return
	}
	head := make(util.M)
	err = buffer.RMAny(head)
	if err != nil {
		return
	}
	json, err := buffer.RBool()
	if err != nil {
		return
	}
	payload := buffer.RAvailable()
	return pkg.InitWithBytes(HdRequest, tid, head, json, payload)
}

func (p *packer) PackResponseOk(tid int64, pkt []byte) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(256)
	buffer.WUint8(HdOk)
	buffer.WInt64(tid)
	_, e := buffer.Write(pkt)
	if e != nil {
		return nil, util.NewErr(util.EcWriteFail, util.M{
			"error": e,
		})
	}
	return buffer.All(), nil
}

func (p *packer) UnpackResponseOk(bytes []byte) (tid int64, payload []byte, err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err = buffer.RInt64()
	if err != nil {
		return
	}
	payload = buffer.RAvailable()
	return
}

func (p *packer) PackResponseFail(tid int64, code uint16) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(13)
	buffer.WUint8(HdFail)
	buffer.WInt64(tid)
	buffer.WUint16(code)
	return buffer.All(), nil
}

func (p *packer) UnpackResponseFail(bytes []byte) (tid int64, code uint16, err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err = buffer.RInt64()
	if err != nil {
		return
	}
	code, err = buffer.RUint16()
	return
}

func (p *packer) PackNotify(tid int64, ntf kiwi.ISndNotice) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(256)
	buffer.WUint8(HdNotify)
	buffer.WInt64(kiwi.GetNodeMeta().NodeId)
	buffer.WInt64(tid)
	err := buffer.WMAny(ntf.Head())
	if err != nil {
		return nil, err
	}
	buffer.WBool(ntf.Json())
	_, e := buffer.Write(ntf.Payload())
	if e != nil {
		return nil, util.NewErr(util.EcWriteFail, util.M{
			"error": e,
		})
	}
	return buffer.All(), nil
}

func (p *packer) UnpackNotify(bytes []byte, pkg kiwi.IRcvNotice) (err *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	buffer.SetPos(1)
	tid, err := buffer.RInt64()
	if err != nil {
		return
	}
	head := make(util.M)
	err = buffer.RMAny(head)
	if err != nil {
		return
	}
	json, err := buffer.RBool()
	if err != nil {
		return
	}
	bs := buffer.RAvailable()
	return pkg.InitWithBytes(HdNotify, tid, head, json, bs)
}

func (p *packer) PackM(m util.M) ([]byte, *util.Err) {
	var buffer util.ByteBuffer
	buffer.InitCap(256)
	err := buffer.WMAny(m)
	if err != nil {
		return nil, err
	}
	return buffer.All(), nil
}

func (p *packer) UnpackM(bytes []byte, m util.M) *util.Err {
	var buffer util.ByteBuffer
	buffer.InitBytes(bytes)
	return buffer.RMAny(m)
}
