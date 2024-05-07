package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

func newNodeBase() nodeBase {
	return nodeBase{}
}

type nodeBase struct {
}

func (n *nodeBase) Init() *util.Err {
	return nil
}

func (n *nodeBase) Connect(ip string, port int, svc kiwi.TSvc, nodeId int64, ver string, head util.M) {
}

func (n *nodeBase) Disconnect(svc kiwi.TSvc, id int64) {
}

func (n *nodeBase) Push(pus kiwi.ISndPush) {
	panic("implement me")
}

func (n *nodeBase) PushNode(nodeId int64, pus kiwi.ISndPush) {
	panic("implement me")
}

func (n *nodeBase) Request(req kiwi.ISndRequest) {
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

func (n *nodeBase) RequestNode(nodeId int64, req kiwi.ISndRequest) {
	panic("implement me")
}

func (n *nodeBase) Notify(ntc kiwi.ISndNotice, filter util.MToBool) {
	if !kiwi.Router().HasNoticeWatcher(ntc.Svc(), ntc.Code()) {
		return
	}
	pkt := NewRcvNtfPkt()
	msg := ntc.Msg()
	if msg != nil {
		pkt.InitWithMsg(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Msg())
	} else {
		err := pkt.InitWithBytes(HdNotify, ntc.Tid(), ntc.Head(), ntc.Json(), ntc.Payload())
		if err != nil {
			kiwi.Error(err)
			return
		}
	}
	kiwi.Router().OnNotice(pkt)
}

func (n *nodeBase) ReceiveWatchNotice(nodeId int64, codes []kiwi.TCode, meta util.M) {
	panic("implement me")
}

func (n *nodeBase) SendToNode(nodeId int64, bytes []byte, fnErr util.FnErr) {
	//TODO implement me
	panic("implement me")
}

func (n *nodeBase) receive(agent kiwi.IAgent, bytes []byte) {
	switch bytes[0] {
	case HdPush:
		n.onPush(agent, bytes)
	case HdRequest:
		n.onRequest(agent, bytes)
	case HdOk:
		n.onResponseOk(agent, bytes)
	case HdFail:
		n.onResponseFail(agent, bytes)
	case HdHeartbeat:
		n.onHeartbeat(agent, bytes)
	case HdNotify:
		n.onNotify(agent, bytes)
	case HdWatch:
		n.onWatchNotify(agent, bytes)
	default:
		kiwi.Error2(util.EcNotExist, util.M{
			"head": bytes[0],
		})
	}
}

func (n *nodeBase) onHeartbeat(agent kiwi.IAgent, bytes []byte) {

}

func (n *nodeBase) onPush(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvPusPkt()
	err := kiwi.Packer().UnpackPush(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnPush(pkt)
}

func (n *nodeBase) onRequest(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvReqPkt()
	err := kiwi.Packer().UnpackRequest(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnRequest(pkt)
}

func (n *nodeBase) onResponseOk(agent kiwi.IAgent, bytes []byte) {
	head := make(util.M)
	tid, payload, err := kiwi.Packer().UnpackResponseOk(bytes, head)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnResponseOkBytes(tid, head, payload)
}

func (n *nodeBase) onResponseFail(agent kiwi.IAgent, bytes []byte) {
	head := make(util.M)
	tid, code, err := kiwi.Packer().UnpackResponseFail(bytes, head)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.TE(tid, err)
		return
	}
	kiwi.Router().OnResponseFail(tid, head, code)
}

func (n *nodeBase) onNotify(agent kiwi.IAgent, bytes []byte) {
	pkt := NewRcvNtfPkt()
	err := kiwi.Packer().UnpackNotify(bytes, pkt)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Router().OnNotice(pkt)
}

func (n *nodeBase) onWatchNotify(agent kiwi.IAgent, bytes []byte) {
	meta := util.M{}
	nodeId, codes, err := kiwi.Packer().UnpackWatchNotify(bytes, meta)
	if err != nil {
		if agent != nil {
			err.AddParam("addr", agent.Addr())
		}
		kiwi.Error(err)
		return
	}
	kiwi.Node().ReceiveWatchNotice(nodeId, codes, meta)
}
