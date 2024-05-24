package kiwi

import (
	"github.com/15mga/kiwi/util"
)

type (
	TSvc          = uint16
	TCode         = uint16
	TSvcCode      = uint16
	NotifyHandler func(pkt IRcvNotice)
	PacketToStr   func(IRcvPkt) string
)

var (
	_Node INode
)

func Node() INode {
	return _Node
}

func SetNode(node INode) {
	_Node = node
}

type INode interface {
	Init()
	Ip() string
	Port() int
	Connect(ip string, port int, svc TSvc, nodeId int64, ver string, head util.M)
	Disconnect(svc TSvc, id int64)
	Push(pus ISndPush)
	PushNode(nodeId int64, pus ISndPush)
	Request(req ISndRequest)
	RequestNode(nodeId int64, req ISndRequest)
	// Notify 广播给所有符合的
	Notify(ntc ISndNotice, filter util.MToBool)
	// NotifyOne 只广播给一个符合的
	NotifyOne(ntc ISndNotice, filter util.MToBool)
	ReceiveWatchNotice(nodeId int64, methods []TCode, meta util.M)
	SendToNode(nodeId int64, bytes []byte, fnErr util.FnErr)
}

type INodeHandler interface {
	Receive(agent IAgent, bytes []byte)
}

type NodeDialerToBool func(INodeDialer) bool

type INodeDialer interface {
	Svc() TSvc
	NodeId() int64
	Dialer() IDialer
	Head() util.M
	Send(bytes []byte, fnErr util.FnErr)
}
