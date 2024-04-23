package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

func InitNodeLocal() {
	kiwi.SetNode(NewNodeLocal())
}

func NewNodeLocal() kiwi.INode {
	return &nodeLocal{
		nodeBase: newNodeBase(),
	}
}

type nodeLocal struct {
	nodeBase
}

func (n *nodeLocal) Push(pus kiwi.ISndPush) {
	pkt := NewRcvPusPkt()
	msg := pus.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdPush, pus.Tid(), pus.Head(), pus.Json(), pus.Msg())
	} else {
		err := pkt.InitWithBytes(HdPush, pus.Tid(), pus.Head(), pus.Json(), pus.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnPush(pkt)
}

func (n *nodeLocal) PushNode(nodeId int64, pus kiwi.ISndPush) {
	n.Push(pus)
}

func (n *nodeLocal) Request(req kiwi.ISndRequest) {
	pkt := NewRcvReqPkt()
	msg := req.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdRequest, req.Tid(), req.Head(), req.Json(), req.Msg())
	} else {
		err := pkt.InitWithBytes(HdRequest, req.Tid(), req.Head(), req.Json(), req.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnRequest(pkt)
}

func (n *nodeLocal) RequestNode(nodeId int64, req kiwi.ISndRequest) {
	n.Request(req)
}

func (n *nodeLocal) Notify(ntf kiwi.ISndNotice, filter util.MToBool) {
	pkt := NewRcvNtfPkt()
	msg := ntf.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdNotify, ntf.Tid(), ntf.Head(), ntf.Json(), ntf.Msg())
	} else {
		err := pkt.InitWithBytes(HdNotify, ntf.Tid(), ntf.Head(), ntf.Json(), ntf.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnNotice(pkt)
}

func (n *nodeLocal) ReceiveWatchNotice(nodeId int64, codes []kiwi.TCode, meta util.M) {
}
