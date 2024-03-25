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
	g.worker = worker.NewJobWorker(g.process)
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
	worker      *worker.JobWorker
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
	g.worker.Push(gateConnected, agent)
}

func (g *gate) onAgentClosed(agent kiwi.IAgent, err *util.Err) {
	g.worker.Push(gateDisconnected, agent, err)
}

func (g *gate) Send(tid int64, id string, bytes []byte, handler util.FnBool) {
	g.worker.Push(gateSend, tid, id, bytes, handler)
}

func (g *gate) AddrSend(tid int64, addr string, bytes []byte, handler util.FnBool) {
	g.worker.Push(gateAddrSend, tid, addr, bytes, handler)
}

func (g *gate) MultiSend(tid int64, idToPayload map[string][]byte, handler util.FnMapBool) {
	g.worker.Push(gateMultiSend, tid, idToPayload, handler)
}

func (g *gate) MultiAddrSend(tid int64, addrToPayload map[string][]byte, handler util.FnMapBool) {
	g.worker.Push(gateMultiAddrSend, tid, addrToPayload, handler)
}

func (g *gate) AllSend(tid int64, bytes []byte) {
	g.worker.Push(gateAllSend, tid, bytes)
}

func (g *gate) UpdateHeadCache(tid int64, id string, head, cache util.M, handler util.FnBool) {
	g.worker.Push(gateUpdate, tid, id, head, cache, handler)
}

func (g *gate) UpdateAddrHeadCache(tid int64, addr string, head, cache util.M, handler util.FnBool) {
	g.worker.Push(gateUpdateAddr, tid, addr, head, cache, handler)
}

func (g *gate) RemoveHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool) {
	g.worker.Push(gateRemove, tid, addr, head, cache, handler)
}

func (g *gate) RemoveAddrHeadCache(tid int64, addr string, head, cache []string, handler util.FnBool) {
	g.worker.Push(gateRemoveAddr, tid, addr, head, cache, handler)
}

func (g *gate) GetHeadCache(tid int64, id string, fn util.FnM2Bool) {
	g.worker.Push(gateGet, tid, id, fn)
}

func (g *gate) GetAddrHeadCache(tid int64, id string, fn util.FnM2Bool) {
	g.worker.Push(gateGetAddr, tid, id, fn)
}

func (g *gate) CloseWithId(tid int64, id string, removeHeadKeys, removeCacheKeys []string) {
	g.worker.Push(gateClose, tid, id, removeHeadKeys, removeCacheKeys)
}

func (g *gate) CloseWithAddr(tid int64, addr string, removeHeadKeys, removeCacheKeys []string) {
	g.worker.Push(gateAddrClose, tid, addr, removeHeadKeys, removeCacheKeys)
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

func (g *gate) process(job *worker.Job) {
	switch job.Name {
	case gateConnected:
		agent := util.SplitSlc1[kiwi.IAgent](job.Data)
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
	case gateDisconnected:
		agent, err := util.SplitSlc2[kiwi.IAgent, *util.Err](job.Data)
		agentAddr := agent.Addr()
		agent, ok := g.addrToAgent.Del(agentAddr)
		if !ok {
			return
		}
		agentId := agent.Id()
		kiwi.Info("agent disconnected", util.M{
			"id":   agentId,
			"addr": agentAddr,
		})
		atomic.AddInt32(&g.agentCount, -1)
		g.option.disconnected(agent, err)
		agent2, ok := g.idToAgent.Get(agentId)
		if !ok || agent2.Addr() != agentAddr { //id被新agent替换
			return
		}
		_, _ = g.idToAgent.Del(agentId)
	case gateSend:
		tid, id, bytes, fn := util.SplitSlc4[int64, string, []byte, util.FnBool](job.Data)
		agent, ok := g.idToAgent.Get(id)
		if !ok {
			fn(false)
			return
		}
		err := agent.Send(bytes)
		if err != nil {
			err.AddParam("id", id)
			kiwi.TE(tid, err)
			fn(false)
			return
		}
		fn(true)
	case gateAddrSend:
		tid, addr, bytes, fn := util.SplitSlc4[int64, string, []byte, util.FnBool](job.Data)
		agent, ok := g.addrToAgent.Get(addr)
		if !ok {
			fn(false)
			return
		}
		err := agent.Send(bytes)
		if err != nil {
			err.AddParam("addr", addr)
			kiwi.TE(tid, err)
			fn(false)
			return
		}
		fn(true)
	case gateMultiSend:
		tid, idToPayload, fn := util.SplitSlc3[int64, map[string][]byte, util.FnMapBool](job.Data)
		m := make(map[string]bool, len(idToPayload))
		for id, payload := range idToPayload {
			agent, ok := g.idToAgent.Get(id)
			if !ok {
				m[id] = false
				continue
			}
			err := agent.Send(payload)
			if err != nil {
				kiwi.TE(tid, err)
				m[id] = false
				continue
			}
			m[id] = true
		}
		fn(m)
	case gateMultiAddrSend:
		tid, addrToPayload, fn := util.SplitSlc3[int64, map[string][]byte, util.FnMapBool](job.Data)
		m := make(map[string]bool, len(addrToPayload))
		for addr, payload := range addrToPayload {
			agent, ok := g.addrToAgent.Get(addr)
			if !ok {
				m[addr] = false
				continue
			}
			err := agent.Send(payload)
			if err != nil {
				kiwi.TE(tid, err)
				m[addr] = false
				continue
			}
			m[addr] = true
		}
		fn(m)
	case gateAllSend:
		tid, bytes := util.SplitSlc2[int64, []byte](job.Data)
		g.idToAgent.Iter(func(item kiwi.IAgent) {
			err := item.Send(util.CopyBytes(bytes))
			if err != nil {
				kiwi.TE(tid, err)
			}
		})
		util.RecycleBytes(bytes)
	case gateUpdate:
		tid, id, head, cache, fn := util.SplitSlc5[int64, string, util.M, util.M, util.FnBool](job.Data)
		agent, ok := g.idToAgent.Get(id)
		if !ok {
			fn(false)
			return
		}
		if head != nil {
			_, ok := head["addr"]
			if ok {
				delete(head, "addr") //这个不能覆盖
			}
			newId, ok := util.MGet[string](head, "id")
			if ok {
				oldId := agent.Id()
				agent.SetId(newId)
				g.idToAgent.ReplaceOrNew(oldId, agent)
			}
			agent.SetHeads(head)
		}
		if cache != nil {
			agent.SetCaches(cache)
		}
		kiwi.TD(tid, "gate id update", util.M{
			"head":  head,
			"cache": cache,
		})
		fn(true)
	case gateUpdateAddr:
		tid, addr, head, cache, fn := util.SplitSlc5[int64, string, util.M, util.M, util.FnBool](job.Data)
		agent, ok := g.addrToAgent.Get(addr)
		if !ok {
			fn(false)
			return
		}
		if head != nil {
			_, ok := head["addr"]
			if ok {
				delete(head, "addr") //这个不能覆盖
			}
			newId, ok := util.MGet[string](head, "id")
			if ok {
				oldId := agent.Id()
				agent.SetId(newId)
				g.idToAgent.ReplaceOrNew(oldId, agent)
			}
			agent.SetHeads(head)
		}
		if cache != nil {
			agent.SetCaches(cache)
		}
		kiwi.TD(tid, "gate addr update", util.M{
			"head":  head,
			"cache": cache,
		})
		fn(true)
	case gateRemove:
		tid, id, head, cache, fn := util.SplitSlc5[int64, string, []string, []string, util.FnBool](job.Data)
		agent, ok := g.idToAgent.Get(id)
		if !ok {
			fn(false)
			return
		}
		agent.DelHead(head...)
		agent.DelCache(cache...)
		kiwi.TD(tid, "gate id remove", util.M{
			"head":  head,
			"cache": cache,
		})
		fn(true)
	case gateRemoveAddr:
		tid, addr, head, cache, fn := util.SplitSlc5[int64, string, []string, []string, util.FnBool](job.Data)
		agent, ok := g.addrToAgent.Get(addr)
		if !ok {
			fn(false)
			return
		}
		agent.DelHead(head...)
		agent.DelCache(cache...)
		kiwi.TD(tid, "gate addr remove", util.M{
			"head":  head,
			"cache": cache,
		})
		fn(true)
	case gateGet:
		_, id, fn := util.SplitSlc3[int64, string, util.FnM2Bool](job.Data)
		agent, ok := g.idToAgent.Get(id)
		if !ok {
			fn(nil, nil, false)
			return
		}
		head := util.M{}
		cache := util.M{}
		agent.CopyHead(head)
		agent.CopyCache(cache)
		fn(head, cache, true)
	case gateGetAddr:
		_, addr, fn := util.SplitSlc3[int64, string, util.FnM2Bool](job.Data)
		agent, ok := g.addrToAgent.Get(addr)
		if !ok {
			fn(nil, nil, false)
			return
		}
		head := util.M{}
		cache := util.M{}
		agent.CopyHead(head)
		agent.CopyCache(cache)
		fn(head, cache, true)
	case gateClose:
		_, id, head, cache := util.SplitSlc4[int64, string, []string, []string](job.Data)
		agent, ok := g.idToAgent.Get(id)
		if !ok {
			return
		}
		agent.DelHead(head...)
		agent.DelCache(cache...)
		agent.Dispose()
	case gateAddrClose:
		_, addr, head, cache := util.SplitSlc4[int64, string, []string, []string](job.Data)
		agent, ok := g.addrToAgent.Get(addr)
		if !ok {
			return
		}
		agent.DelHead(head...)
		agent.DelCache(cache...)
		agent.Dispose()
	}
}

const (
	gateConnected     = "connected"
	gateDisconnected  = "disconnected"
	gateSend          = "send"
	gateAddrSend      = "send_addr"
	gateMultiSend     = "multi_send"
	gateMultiAddrSend = "multi_send_addr"
	gateAllSend       = "all_send"
	gateUpdate        = "update_head_cache"
	gateUpdateAddr    = "update_addr_head_cache"
	gateRemove        = "remove_head_cache"
	gateRemoveAddr    = "remove_addr_head_cache"
	gateGet           = "get_head_cache"
	gateGetAddr       = "get_addr_head_cache"
	gateAddrClose     = "addr_close"
	gateClose         = "close"
)
