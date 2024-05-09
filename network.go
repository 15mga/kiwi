package kiwi

import (
	"context"
	"github.com/15mga/kiwi/util"
)

type (
	FnAgent              func(IAgent)
	FnAgentBool          func(IAgent, bool)
	FnAgentErr           func(IAgent, *util.Err)
	FnAgentBytes         func(IAgent, []byte)
	AgentBytesToBytesErr func(IAgent, []byte) ([]byte, *util.Err)
	AgentBytesToBytes    func(IAgent, []byte) []byte
)

// IAgent 连接代理接口
type IAgent interface {
	SetHead(key string, val any)
	SetHeads(m util.M)
	GetHead(key string) (any, bool)
	DelHead(keys ...string)
	ClearHead()
	CopyHead(m util.M)
	SetCache(key string, val any)
	SetCaches(m util.M)
	GetCache(key string) (any, bool)
	DelCache(keys ...string)
	CopyCache(m util.M)
	ClearCache()
	Id() string
	SetId(id string)
	Addr() string
	Host() string
	// Enable 状态
	Enable() *util.Enable
	// Send 发送数据
	Send(bytes []byte) *util.Err
	// Dispose 释放
	Dispose()
	BindConnected(fn FnAgent)
	BindDisconnected(fn FnAgentErr)
}

// IListener 监听器
type IListener interface {
	// Addr 监听地址
	Addr() string
	Port() int
	// Start 开始监听
	Start() *util.Err
	// Close 关闭监听
	Close()
}

// IDialer 拨号器接口
type IDialer interface {
	// Name 拨号器名称
	Name() string
	// Connect 连接远程服务器
	Connect(ctx context.Context) *util.Err
	// Agent 连接代理
	Agent() IAgent
}

type AgentRWMode uint8

const (
	AgentRW AgentRWMode = iota
	AgentR
	AgentW
)

// AgentOpt Agent代理选项
type (
	AgentOpt struct {
		PacketMaxCap int        //最大包长
		PacketMinCap int        //包长最小容量
		OnErr        util.FnErr //处理错误
		DeadlineSecs int
		AgentMode    AgentRWMode
		HeadLen      int
	}
	AgentOption func(o *AgentOpt)
)

// AgentPacketMaxCap 最大包长
func AgentPacketMaxCap(packetMaxCap int) AgentOption {
	return func(o *AgentOpt) {
		o.PacketMaxCap = packetMaxCap
	}
}

// AgentPacketMinCap 包长最小容量
func AgentPacketMinCap(packetMinCap int) AgentOption {
	return func(o *AgentOpt) {
		o.PacketMinCap = packetMinCap
	}
}

// AgentErr 连接关闭的回调,Error为空正常关闭
func AgentErr(onErr util.FnErr) AgentOption {
	return func(o *AgentOpt) {
		o.OnErr = onErr
	}
}

func AgentDeadline(secs int) AgentOption {
	return func(o *AgentOpt) {
		o.DeadlineSecs = secs
	}
}

func AgentMode(mode AgentRWMode) AgentOption {
	return func(o *AgentOpt) {
		o.AgentMode = mode
	}
}

func AgentHeadLen(length int) AgentOption {
	return func(o *AgentOpt) {
		o.HeadLen = length
	}
}
