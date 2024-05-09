package network

import (
	"context"
	"github.com/15mga/kiwi"
	"math"
	"net"
	"sync"

	"github.com/15mga/kiwi/ds"

	"github.com/15mga/kiwi/util"
)

const (
	_PacketMaxCap int = math.MaxUint32
	_PacketMinCap int = 1 << 11
)

func newAgent(addr string, receiver kiwi.FnAgentBytes, opts ...kiwi.AgentOption) agent {
	opt := &kiwi.AgentOpt{
		PacketMaxCap: _PacketMaxCap,
		PacketMinCap: _PacketMinCap,
		HeadLen:      4,
	}
	for _, action := range opts {
		action(opt)
	}
	a := agent{
		id:             addr,
		addr:           addr,
		option:         opt,
		enable:         util.NewEnable(),
		receiver:       receiver,
		bytesLink:      ds.NewLink[[]byte](),
		onConnected:    ds.NewFnLink1[kiwi.IAgent](),
		onDisconnected: ds.NewFnLink2[kiwi.IAgent, *util.Err](),
		head:           util.M{},
		cache:          util.M{},
		mtx:            &sync.RWMutex{},
	}
	a.head.Set("addr", addr)
	return a
}

type agent struct {
	option         *kiwi.AgentOpt
	onClose        func() error
	id             string
	addr           string
	ctx            context.Context
	cancel         context.CancelFunc
	writeSignCh    chan struct{}
	enable         *util.Enable
	receiver       kiwi.FnAgentBytes
	bytesLink      *ds.Link[[]byte]
	onConnected    *ds.FnLink1[kiwi.IAgent]
	onDisconnected *ds.FnLink2[kiwi.IAgent, *util.Err]
	head           util.M
	cache          util.M
	mtx            *sync.RWMutex
}

func (a *agent) onStart(_ []any) {
	a.writeSignCh = make(chan struct{}, 1)
	go a.onConnected.Invoke(a)
}

func (a *agent) start(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	_ = a.enable.Enable(a.onStart)
}

func (a *agent) SetHead(key string, val any) {
	a.mtx.Lock()
	a.head[key] = val
	a.mtx.Unlock()
}

func (a *agent) SetHeads(m util.M) {
	a.mtx.Lock()
	for key, val := range m {
		a.head[key] = val
	}
	a.mtx.Unlock()
}

func (a *agent) GetHead(key string) (val any, exist bool) {
	a.mtx.RLock()
	val, exist = a.head[key]
	a.mtx.RUnlock()
	return
}

func (a *agent) DelHead(keys ...string) {
	a.mtx.Lock()
	for _, key := range keys {
		delete(a.head, key)
	}
	a.mtx.Unlock()
}

func (a *agent) ClearHead() {
	a.mtx.Lock()
	a.head = util.M{}
	a.mtx.Unlock()
}

func (a *agent) CopyHead(m util.M) {
	if m == nil {
		return
	}
	a.mtx.Lock()
	for key, val := range a.head {
		m[key] = val
	}
	a.mtx.Unlock()
}

func (a *agent) SetCache(key string, val any) {
	a.mtx.Lock()
	a.cache[key] = val
	a.mtx.Unlock()
}

func (a *agent) SetCaches(m util.M) {
	a.mtx.Lock()
	for key, val := range m {
		a.cache[key] = val
	}
	a.mtx.Unlock()
}

func (a *agent) GetCache(key string) (val any, exist bool) {
	a.mtx.RLock()
	val, exist = a.cache[key]
	a.mtx.RUnlock()
	return
}

func (a *agent) DelCache(keys ...string) {
	a.mtx.Lock()
	for _, key := range keys {
		delete(a.cache, key)
	}
	a.mtx.Unlock()
}

func (a *agent) CopyCache(m util.M) {
	a.mtx.Lock()
	for key, val := range a.cache {
		m[key] = val
	}
	a.mtx.Unlock()
}

func (a *agent) ClearCache() {
	a.mtx.Lock()
	a.cache = util.M{}
	a.mtx.Unlock()
}

func (a *agent) Id() (id string) {
	a.mtx.RLock()
	id = a.id
	a.mtx.RUnlock()
	return id
}

func (a *agent) SetId(id string) {
	a.mtx.Lock()
	a.id = id
	a.mtx.Unlock()
}

func (a *agent) Addr() string {
	return a.addr
}

func (a *agent) Host() string {
	host, _, _ := net.SplitHostPort(a.addr)
	return host
}

func (a *agent) Enable() *util.Enable {
	return a.enable
}

func (a *agent) Dispose() {
	a.cancel()
}

func (a *agent) Send(bytes []byte) *util.Err {
	if bytes == nil {
		return util.NewErr(util.EcEmpty, nil)
	}
	l := len(bytes)
	if l == 0 {
		return util.NewErr(util.EcEmpty, nil)
	}
	if l > a.option.PacketMaxCap {
		return util.NewErr(util.EcTooLong, util.M{
			"length": l,
		})
	}
	return a.enable.WAction(agentPushByte, a, bytes)
}

func agentPushByte(params []any) {
	a, bytes := util.SplitSlc2[*agent, []byte](params)
	a.bytesLink.Push(bytes)
	select {
	case a.writeSignCh <- struct{}{}:
	default:
	}
}

func (a *agent) BindConnected(fn kiwi.FnAgent) {
	a.onConnected.Push(fn)
}

func (a *agent) BindDisconnected(fn kiwi.FnAgentErr) {
	a.onDisconnected.Push(fn)
}

func (a *agent) close(err *util.Err) {
	a.enable.Disable(agentClose, a, err)
}

func agentClose(params []any) {
	a, err := util.SplitSlc2[*agent, *util.Err](params)
	close(a.writeSignCh)
	_ = a.onClose()
	go a.onDisconnected.Invoke(a, err)
}
