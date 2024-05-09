package kiwi

import (
	"github.com/15mga/kiwi/util"
)

var (
	_Gate IGate
)

func Gate() IGate {
	return _Gate
}

func SetGate(gate IGate) {
	_Gate = gate
}

type (
	GateReceiver func(agent IAgent, svc, method string, head util.M, body []byte, fnErr util.FnErr)
)

type IGate interface {
	Dispose() *util.Err
	Send(tid int64, id string, bytes []byte, handler util.FnBool)
	AddrSend(tid int64, addr string, bytes []byte, handler util.FnBool)
	MultiSend(tid int64, idToPayload map[string][]byte, handler util.FnMapBool)
	MultiAddrSend(tid int64, addrToPayload map[string][]byte, handler util.FnMapBool)
	AllSend(tid int64, bytes []byte)
	// CloseWithId 关闭并清理会话
	CloseWithId(tid int64, id string)
	// CloseWithAddr 关闭并清理会话
	CloseWithAddr(tid int64, addr string)
	// DisconnectAll 断开所有链接，不清空会话，允许发起重连，用于网关主动断开
	DisconnectAll(tid int64)
	UpdateHeadCache(tid int64, id string, head, cache util.M, handler util.FnBool)
	UpdateAddrHeadCache(tid int64, addr string, head, cache util.M, handler util.FnBool)
	RemoveHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool)
	RemoveAddrHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool)
	GetHeadCache(tid int64, id string, fn util.FnM2Bool)
	GetAddrHeadCache(tid int64, id string, fn util.FnM2Bool)
	SetRoles(m map[TSvcCode][]int64)
	Authenticate(mask int64, svc TSvc, code TCode) bool
}
