package core

import (
	"fmt"
	"github.com/15mga/kiwi"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/network"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"github.com/fasthttp/websocket"
)

const (
	DefConnCap = 1 << 12
)

type (
	GateOption func(option *gateOption)
	gateOption struct {
		ip           string
		tcp          int
		web          int
		webOpts      []network.WebOption
		udp          int
		connCap      int32
		connected    kiwi.FnAgent
		disconnected kiwi.FnAgentErr
		checkIp      util.StrToBool
		deadline     int
		headLen      int
		roles        map[kiwi.TSvcCode][]int64
	}
)

func GateIp(ip string) GateOption {
	return func(option *gateOption) {
		option.ip = ip
	}
}

func GateTcpPort(port int) GateOption {
	return func(option *gateOption) {
		option.tcp = port
	}
}

func GateWebsocketPort(port int) GateOption {
	return func(option *gateOption) {
		option.web = port
	}
}

func GateWebsocketOptions(opts ...network.WebOption) GateOption {
	return func(option *gateOption) {
		option.webOpts = opts
	}
}

func GateUdpPort(port int) GateOption {
	return func(option *gateOption) {
		option.udp = port
	}
}

func GateConnCap(cap int32) GateOption {
	return func(option *gateOption) {
		option.connCap = cap
	}
}

func GateConnected(connected kiwi.FnAgent) GateOption {
	return func(option *gateOption) {
		option.connected = connected
	}
}

func GateDisconnected(disconnected kiwi.FnAgentErr) GateOption {
	return func(option *gateOption) {
		option.disconnected = disconnected
	}
}

func GateCheckIp(fn util.StrToBool) GateOption {
	return func(option *gateOption) {
		option.checkIp = fn
	}
}

func GateDeadlineSecs(deadline int) GateOption {
	return func(option *gateOption) {
		option.deadline = deadline
	}
}

func GateHeadLen(headLen int) GateOption {
	return func(option *gateOption) {
		option.headLen = headLen
	}
}

func GateRoles(roles map[kiwi.TSvcCode][]int64) GateOption {
	return func(option *gateOption) {
		option.roles = roles
	}
}

func InitGate(receiver kiwi.FnAgentBytes, opts ...GateOption) {
	o := &gateOption{
		connCap: DefConnCap,
		checkIp: func(s string) bool {
			return true
		},
		headLen: 4,
		connected: func(agent kiwi.IAgent) {

		},
		disconnected: func(agent kiwi.IAgent, err *util.Err) {

		},
	}
	for _, opt := range opts {
		opt(o)
	}
	g := &gate{
		option:   o,
		receiver: receiver,
		idToAgent: ds.NewKSet[string, kiwi.IAgent](1024, func(agent kiwi.IAgent) string {
			return agent.Id()
		}),
		addrToAgent: ds.NewKSet[string, kiwi.IAgent](1024, func(agent kiwi.IAgent) string {
			return agent.Addr()
		}),
		msgToRoles: sync.Map{},
	}
	g.SetRoles(o.roles)
	g.worker = worker.NewWorker(4096, g.process)
	g.worker.Start()
	if o.ip == "" {
		ip, err := util.GetLocalIp()
		if err != nil {
			kiwi.Fatal(err)
		}
		kiwi.Info("use local ip", util.M{
			"ip": ip,
		})
		o.ip = ip
	}
	if g.option.udp > 0 {
		addr := fmt.Sprintf("%s:%d", g.option.ip, g.option.udp)
		listener := network.NewUdpListener(addr, g.onAddUdpConn)
		g.listeners = append(g.listeners, listener)
		err := listener.Start()
		if err != nil {
			kiwi.Fatal(err)
		}
		_ = kiwi.GetNodeMeta().Data.Set2(g.option.udp, "gate", "udp")
	}
	if g.option.tcp > 0 {
		addr := fmt.Sprintf("%s:%d", g.option.ip, g.option.tcp)
		listener := network.NewTcpListener(addr, g.onAddTcpConn)
		err := listener.Start()
		if err != nil {
			kiwi.Fatal(err)
		}
		g.listeners = append(g.listeners, listener)
		_ = kiwi.GetNodeMeta().Data.Set2(g.option.tcp, "gate", "tcp")
	}
	if g.option.web > 0 {
		addr := fmt.Sprintf("%s:%d", g.option.ip, g.option.web)
		listener := network.NewWebListener(g.onAddWebConn, append(g.option.webOpts, network.WebAddr(addr))...)
		err := listener.Start()
		if err != nil {
			kiwi.Fatal(err)
		}
		g.listeners = append(g.listeners, listener)
		_ = kiwi.GetNodeMeta().Data.Set2(g.option.web, "gate", "web")
	}
	kiwi.SetGate(g)
}

type gate struct {
	option      *gateOption
	receiver    kiwi.FnAgentBytes
	worker      *worker.Worker
	listeners   []kiwi.IListener
	idToAgent   *ds.KSet[string, kiwi.IAgent]
	addrToAgent *ds.KSet[string, kiwi.IAgent]
	agentCount  int32
	msgToRoles  sync.Map
}

func (g *gate) Dispose() *util.Err {
	for _, listener := range g.listeners {
		listener.Close()
	}
	return nil
}

func (g *gate) onAddTcpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	c := atomic.LoadInt32(&g.agentCount)
	if c == g.option.connCap {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcTooManyConn, util.M{
			"addr": addr,
		}))
		return
	}

	if !g.option.checkIp(strings.Split(addr, ":")[0]) {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcIllegalConn, util.M{
			"addr": addr,
		}))
		return
	}

	agent := network.NewTcpAgent(addr, g.receiver,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentDeadline(g.option.deadline),
		kiwi.AgentHeadLen(g.option.headLen),
	)
	agent.BindConnected(g.onAgentConnected)
	agent.BindDisconnected(g.onAgentClosed)
	agent.Start(util.Ctx(), conn)
}

func (g *gate) onAddUdpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	c := atomic.LoadInt32(&g.agentCount)
	if c == g.option.connCap {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcTooManyConn, util.M{
			"addr": addr,
		}))
		return
	}

	if !g.option.checkIp(strings.Split(addr, ":")[0]) {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcIllegalConn, util.M{
			"addr": addr,
		}))
		return
	}

	agent := network.NewUdpAgent(addr, g.receiver,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentDeadline(g.option.deadline),
		kiwi.AgentHeadLen(g.option.headLen),
	)
	agent.BindConnected(g.onAgentConnected)
	agent.BindDisconnected(g.onAgentClosed)
	agent.Start(util.Ctx(), conn)
}

func (g *gate) onAddWebConn(conn *websocket.Conn) {
	addr := conn.RemoteAddr().String()
	c := atomic.LoadInt32(&g.agentCount)
	if c == g.option.connCap {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcTooManyConn, util.M{
			"addr": addr,
		}))
		return
	}

	if !g.option.checkIp(strings.Split(addr, ":")[0]) {
		_ = conn.Close()
		kiwi.Warn(util.NewErr(util.EcIllegalConn, util.M{
			"addr": addr,
		}))
		return
	}

	agent := network.NewWebAgent(addr, 2, g.receiver,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentDeadline(g.option.deadline),
		kiwi.AgentHeadLen(g.option.headLen),
	)
	agent.BindConnected(g.onAgentConnected)
	agent.BindDisconnected(g.onAgentClosed)
	agent.Start(util.Ctx(), conn)
}

func (g *gate) onAgentConnected(agent kiwi.IAgent) {
	g.worker.Push(gateJobConnected{agent})
}

func (g *gate) onAgentClosed(agent kiwi.IAgent, err *util.Err) {
	g.worker.Push(gateJobDisconnected{agent, err})
}

func (g *gate) Send(tid int64, id string, bytes []byte, handler util.FnBool) {
	g.worker.Push(gateJobSend{tid, id, bytes, handler})
}

func (g *gate) AddrSend(tid int64, addr string, bytes []byte, handler util.FnBool) {
	g.worker.Push(gateJobAddrSend{tid, addr, bytes, handler})
}

func (g *gate) MultiSend(tid int64, idToPayload map[string][]byte, handler util.FnMapBool) {
	g.worker.Push(gateJobMultiSend{tid, idToPayload, handler})
}

func (g *gate) MultiAddrSend(tid int64, addrToPayload map[string][]byte, handler util.FnMapBool) {
	g.worker.Push(gateJobMultiAddrSend{tid, addrToPayload, handler})
}

func (g *gate) AllSend(tid int64, bytes []byte) {
	g.worker.Push(gateJobAllSend{tid, bytes})
}

func (g *gate) UpdateHeadCache(tid int64, id string, head, cache util.M, handler util.FnBool) {
	g.worker.Push(gateJobUpdate{tid, id, head, cache, handler})
}

func (g *gate) UpdateAddrHeadCache(tid int64, addr string, head, cache util.M, handler util.FnBool) {
	g.worker.Push(gateJobAddrUpdate{tid, addr, head, cache, handler})
}

func (g *gate) RemoveHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool) {
	g.worker.Push(gateJobRemove{tid, addr, head, cache, handler})
}

func (g *gate) RemoveAddrHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool) {
	g.worker.Push(gateJobAddrRemove{tid, addr, head, cache, handler})
}

func (g *gate) GetHeadCache(tid int64, id string, fn util.FnM2Bool) {
	g.worker.Push(gateJobGet{tid, id, fn})
}

func (g *gate) GetAddrHeadCache(tid int64, id string, fn util.FnM2Bool) {
	g.worker.Push(gateJobAddrGet{tid, id, fn})
}

func (g *gate) CloseWithId(tid int64, id string, removeHeadKeys, removeCacheKeys []string) {
	g.worker.Push(gateJobClose{tid, id, removeHeadKeys, removeCacheKeys})
}

func (g *gate) CloseWithAddr(tid int64, addr string, removeHeadKeys, removeCacheKeys []string) {
	g.worker.Push(gateJobAddrClose{tid, addr, removeHeadKeys, removeCacheKeys})
}

func (g *gate) SetRoles(m map[kiwi.TSvcCode][]int64) {
	for code, roles := range m {
		g.msgToRoles.Store(code, roles)
	}
}

func (g *gate) Authenticate(mask int64, svc kiwi.TSvc, code kiwi.TCode) bool {
	slc, o := g.msgToRoles.Load(kiwi.MergeSvcCode(svc, code))
	if !o {
		return false
	}
	for _, role := range slc.([]int64) {
		if util.TestMask(role, mask) {
			return true
		}
	}
	return false
}

func (g *gate) process(data any) {
	switch d := data.(type) {
	case gateJobConnected:
		agent := d.agent
		ok := g.idToAgent.AddNX(agent)
		if !ok {
			return
		}
		_ = g.addrToAgent.AddNX(agent)
		kiwi.Info("agent connected", util.M{
			"id":   agent.Id(),
			"addr": agent.Addr(),
		})
		atomic.AddInt32(&g.agentCount, 1)
		g.option.connected(agent)
	case gateJobDisconnected:
		agentAddr := d.agent.Addr()
		agent, ok := g.addrToAgent.Del(agentAddr)
		if !ok {
			return
		}
		agentId := d.agent.Id()
		kiwi.Info("agent disconnected", util.M{
			"id":   agentId,
			"addr": agentAddr,
		})
		atomic.AddInt32(&g.agentCount, -1)
		g.option.disconnected(agent, d.err)
		agent2, ok := g.idToAgent.Get(agentId)
		if !ok || agent2.Addr() != agentAddr { //id被新agent替换
			return
		}
		_, _ = g.idToAgent.Del(agentId)
	case gateJobSend:
		agent, ok := g.idToAgent.Get(d.id)
		if !ok {
			d.fn(false)
			return
		}
		err := agent.Send(d.payload)
		if err != nil {
			err.AddParam("id", d.id)
			kiwi.TE(d.tid, err)
			d.fn(false)
			return
		}
		d.fn(true)
	case gateJobAddrSend:
		agent, ok := g.addrToAgent.Get(d.addr)
		if !ok {
			d.fn(false)
			return
		}
		err := agent.Send(d.payload)
		if err != nil {
			err.AddParam("addr", d.addr)
			kiwi.TE(d.tid, err)
			d.fn(false)
			return
		}
		d.fn(true)
	case gateJobMultiSend:
		m := make(map[string]bool, len(d.idToPayload))
		for id, payload := range d.idToPayload {
			agent, ok := g.idToAgent.Get(id)
			if !ok {
				m[id] = false
				continue
			}
			err := agent.Send(payload)
			if err != nil {
				kiwi.TE(d.tid, err)
				m[id] = false
				continue
			}
			m[id] = true
		}
		d.fn(m)
	case gateJobMultiAddrSend:
		m := make(map[string]bool, len(d.addrToPayload))
		for addr, payload := range d.addrToPayload {
			agent, ok := g.addrToAgent.Get(addr)
			if !ok {
				m[addr] = false
				continue
			}
			err := agent.Send(payload)
			if err != nil {
				kiwi.TE(d.tid, err)
				m[addr] = false
				continue
			}
			m[addr] = true
		}
		d.fn(m)
	case gateJobAllSend:
		g.idToAgent.Iter(func(item kiwi.IAgent) {
			err := item.Send(util.CopyBytes(d.payload))
			if err != nil {
				kiwi.TE(d.tid, err)
			}
		})
		util.RecycleBytes(d.payload)
	case gateJobUpdate:
		agent, ok := g.idToAgent.Get(d.id)
		if !ok {
			d.fn(false)
			return
		}
		if d.head != nil {
			_, ok := d.head["addr"]
			if ok {
				delete(d.head, "addr") //这个不能覆盖
			}
			newId, ok := util.MGet[string](d.head, "id")
			if ok {
				oldId := agent.Id()
				agent.SetId(newId)
				g.idToAgent.ReplaceOrNew(oldId, agent)
			}
			agent.SetHeads(d.head)
		}
		if d.cache != nil {
			agent.SetCaches(d.cache)
		}
		kiwi.TD(d.tid, "gate id update", util.M{
			"head":  d.head,
			"cache": d.cache,
		})
		d.fn(true)
	case gateJobAddrUpdate:
		agent, ok := g.addrToAgent.Get(d.addr)
		if !ok {
			d.fn(false)
			return
		}
		if d.head != nil {
			_, ok := d.head["addr"]
			if ok {
				delete(d.head, "addr") //这个不能覆盖
			}
			newId, ok := util.MGet[string](d.head, "id")
			if ok {
				oldId := agent.Id()
				agent.SetId(newId)
				g.idToAgent.ReplaceOrNew(oldId, agent)
			}
			agent.SetHeads(d.head)
		}
		if d.cache != nil {
			agent.SetCaches(d.cache)
		}
		kiwi.TD(d.tid, "gate addr update", util.M{
			"head":  d.head,
			"cache": d.cache,
		})
		d.fn(true)
	case gateJobRemove:
		agent, ok := g.idToAgent.Get(d.id)
		if !ok {
			d.fn(false)
			return
		}
		agent.DelHead(d.head...)
		agent.DelCache(d.cache...)
		kiwi.TD(d.tid, "gate id remove", util.M{
			"head":  d.head,
			"cache": d.cache,
		})
		d.fn(true)
	case gateJobAddrRemove:
		agent, ok := g.addrToAgent.Get(d.addr)
		if !ok {
			d.fn(false)
			return
		}
		agent.DelHead(d.head...)
		agent.DelCache(d.cache...)
		kiwi.TD(d.tid, "gate addr remove", util.M{
			"head":  d.head,
			"cache": d.cache,
		})
		d.fn(true)
	case gateJobGet:
		agent, ok := g.idToAgent.Get(d.id)
		if !ok {
			d.fn(nil, nil, false)
			return
		}
		head := util.M{}
		cache := util.M{}
		agent.CopyHead(head)
		agent.CopyCache(cache)
		d.fn(head, cache, true)
	case gateJobAddrGet:
		agent, ok := g.addrToAgent.Get(d.addr)
		if !ok {
			d.fn(nil, nil, false)
			return
		}
		head := util.M{}
		cache := util.M{}
		agent.CopyHead(head)
		agent.CopyCache(cache)
		d.fn(head, cache, true)
	case gateJobClose:
		agent, ok := g.idToAgent.Get(d.id)
		if !ok {
			return
		}
		agent.DelHead(d.head...)
		agent.DelCache(d.cache...)
		agent.Dispose()
	case gateJobAddrClose:
		agent, ok := g.addrToAgent.Get(d.addr)
		if !ok {
			return
		}
		agent.DelHead(d.head...)
		agent.DelCache(d.cache...)
		agent.Dispose()
	}
}

type gateJobConnected struct {
	agent kiwi.IAgent
}

type gateJobDisconnected struct {
	agent kiwi.IAgent
	err   *util.Err
}

type gateJobSend struct {
	tid     int64
	id      string
	payload []byte
	fn      util.FnBool
}

type gateJobAddrSend struct {
	tid     int64
	addr    string
	payload []byte
	fn      util.FnBool
}

type gateJobMultiSend struct {
	tid         int64
	idToPayload map[string][]byte
	fn          util.FnMapBool
}

type gateJobMultiAddrSend struct {
	tid           int64
	addrToPayload map[string][]byte
	fn            util.FnMapBool
}

type gateJobAllSend struct {
	tid     int64
	payload []byte
}

type gateJobUpdate struct {
	tid   int64
	id    string
	head  util.M
	cache util.M
	fn    util.FnBool
}

type gateJobAddrUpdate struct {
	tid   int64
	addr  string
	head  util.M
	cache util.M
	fn    util.FnBool
}

type gateJobRemove struct {
	tid   int64
	id    string
	head  []string
	cache []string
	fn    util.FnBool
}

type gateJobAddrRemove struct {
	tid   int64
	addr  string
	head  []string
	cache []string
	fn    util.FnBool
}

type gateJobGet struct {
	tid int64
	id  string
	fn  util.FnM2Bool
}

type gateJobAddrGet struct {
	tid  int64
	addr string
	fn   util.FnM2Bool
}

type gateJobClose struct {
	tid   int64
	id    string
	head  []string
	cache []string
}

type gateJobAddrClose struct {
	tid   int64
	addr  string
	head  []string
	cache []string
}
