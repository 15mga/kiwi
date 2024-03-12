package core

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/network"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"math/rand"
	"net"
)

type (
	NodeOption func(opt *nodeOption)
	nodeOption struct {
		ip       string
		port     int
		connType NodeConnType
		selector NodeDialerSelector
	}
	NodeConnType uint8
)

type NodeDialerSelector func(set *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer]) (int64, *util.Err)

const (
	Tcp NodeConnType = iota
	Udp
)

func NodeType(t NodeConnType) NodeOption {
	return func(opt *nodeOption) {
		opt.connType = t
	}
}

func NodeIp(ip string) NodeOption {
	return func(opt *nodeOption) {
		opt.ip = ip
	}
}

func NodePort(port int) NodeOption {
	return func(opt *nodeOption) {
		opt.port = port
	}
}

func NodeSelector(selector NodeDialerSelector) NodeOption {
	return func(opt *nodeOption) {
		opt.selector = selector
	}
}

func InitNodeNet(opts ...NodeOption) {
	kiwi.SetNode(NewNodeNet(opts...))
}

func NewNodeNet(opts ...NodeOption) kiwi.INode {
	opt := &nodeOption{
		connType: Tcp,
		selector: func(set *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer]) (int64, *util.Err) {
			i := rand.Intn(set.Count())
			dialer, _ := set.GetWithIdx(i)
			return dialer.NodeId(), nil
		},
	}
	for _, o := range opts {
		o(opt)
	}
	n := &nodeNet{
		option:   opt,
		nodeBase: newNodeBase(),
		svcToDialer: ds.NewKSet2[kiwi.TSvc, int64, kiwi.INodeDialer](8, func(dialer kiwi.INodeDialer) int64 {
			return dialer.NodeId()
		}),
		idToDialer: ds.NewKSet[int64, kiwi.INodeDialer](16, func(dialer kiwi.INodeDialer) int64 {
			return dialer.NodeId()
		}),
		codeToWatchers: make(map[kiwi.TCode]map[int64]struct{}),
	}
	ip, err := util.CheckLocalIp(opt.ip)
	if err != nil {
		kiwi.Fatal(err)
	}
	opt.ip = ip
	n.worker = worker.NewJobWorker(n.processor)
	addr := fmt.Sprintf("%s:%d", n.option.ip, n.option.port)
	switch opt.connType {
	case Tcp:
		n.listener = network.NewTcpListener(addr, n.onAddTcpConn)
	case Udp:
		n.listener = network.NewUdpListener(addr, n.onAddUdpConn)
	}

	err = n.listener.Start()
	if err != nil {
		panic(err.Error())
	}

	n.worker.Start()
	return n
}

type nodeNet struct {
	nodeBase
	option         *nodeOption
	worker         *worker.JobWorker
	svcToDialer    *ds.KSet2[kiwi.TSvc, int64, kiwi.INodeDialer]
	idToDialer     *ds.KSet[int64, kiwi.INodeDialer]
	listener       kiwi.IListener
	codeToWatchers map[kiwi.TCode]map[int64]struct{} //本机方法的远程监听者
	watcherToCodes map[int64][]kiwi.TCode            //远程机监听的方法
}

func (n *nodeNet) Init() *util.Err {
	return nil
}

func (n *nodeNet) Ip() string {
	return n.option.ip
}

func (n *nodeNet) Port() int {
	return n.listener.Port()
}

func (n *nodeNet) Connect(ip string, port int, svc kiwi.TSvc, nodeId int64, ver string, head util.M) {
	n.worker.Push(nodeConnect, ip, port, svc, nodeId, ver, head)
}

func (n *nodeNet) Disconnect(svc kiwi.TSvc, nodeId int64) {
	n.worker.Push(nodeDisconnect, svc, nodeId)
}

func (n *nodeNet) onConnected(dialer *nodeDialer) {
	n.worker.Push(nodeConnected, dialer)
}

func (n *nodeNet) onDisconnected(dialer *nodeDialer, err *util.Err) {
	n.worker.Push(nodeDisconnected, dialer, err)
}

func (n *nodeNet) pushSelf(pus kiwi.ISndPush) {
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

func (n *nodeNet) Push(pus kiwi.ISndPush) {
	if kiwi.GetNodeMeta().HasService(pus.Svc()) {
		n.pushSelf(pus)
		return
	}
	n.worker.Push(nodePush, pus)
}

func (n *nodeNet) PushNode(nodeId int64, pus kiwi.ISndPush) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.pushSelf(pus)
		return
	}
	n.worker.Push(nodePushNode, nodeId, pus)
}

func (n *nodeNet) requestSelf(req kiwi.ISndRequest) {
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

func (n *nodeNet) Request(req kiwi.ISndRequest) {
	if kiwi.GetNodeMeta().HasService(req.Svc()) {
		n.requestSelf(req)
		return
	}
	n.worker.Push(nodeRequest, req)
}

func (n *nodeNet) RequestNode(nodeId int64, req kiwi.ISndRequest) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.requestSelf(req)
		return
	}
	n.worker.Push(nodeRequestNode, req)
}

func (n *nodeNet) Notify(ntf kiwi.ISndNotice) {
	n.worker.Push(nodeSendNotify, ntf)
}

func (n *nodeNet) ReceiveWatchNotice(nodeId int64, codes []kiwi.TCode) {
	n.worker.Push(nodeWatchNotify, nodeId, codes)
}

func (n *nodeNet) SendToNode(nodeId int64, bytes []byte, fnErr util.FnErr) {
	n.worker.Push(nodeSendNode, nodeId, bytes, fnErr)
}

func (n *nodeNet) onAddTcpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	agent := network.NewTcpAgent(addr, n.receive,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentMode(kiwi.AgentR),
		kiwi.AgentDeadline(30),
	)
	agent.Start(util.Ctx(), conn)
}

func (n *nodeNet) onAddUdpConn(conn net.Conn) {
	addr := conn.RemoteAddr().String()
	agent := network.NewUdpAgent(addr, n.receive,
		kiwi.AgentErr(func(err *util.Err) {
			err.AddParam("addr", addr)
			kiwi.Error(err)
		}),
		kiwi.AgentMode(kiwi.AgentR),
		kiwi.AgentDeadline(30),
	)
	agent.Start(util.Ctx(), conn)
}

func (n *nodeNet) createDialer(name, addr string) kiwi.IDialer {
	switch n.option.connType {
	case Tcp:
		return network.NewTcpDialer(name, addr, n.receive, kiwi.AgentMode(kiwi.AgentW))
	case Udp:
		return network.NewUdpDialer(name, addr, n.receive, kiwi.AgentMode(kiwi.AgentW))
	default:
		kiwi.Fatal2(util.EcParamsErr, util.M{
			"conn type": n.option.connType,
		})
		return nil
	}
}

func (n *nodeNet) processor(job *worker.Job) {
	switch job.Name {
	case nodeConnect:
		ip, port, svc, nodeId, ver, head := util.SplitSlc6[string, int, kiwi.TSvc, int64, string, util.M](job.Data)
		if n.idToDialer.Has(nodeId) {
			kiwi.Info("exist service", util.M{
				"node id": nodeId,
			})
			return
		}
		kiwi.Info("connect service", util.M{
			"ip":      ip,
			"port":    port,
			"svc":     svc,
			"node id": nodeId,
			"ver":     ver,
			"head":    head,
		})
		dialer := n.createDialer(fmt.Sprintf("%s_%d", svc, nodeId), fmt.Sprintf("%s:%d", ip, port))
		newNodeDialer(dialer, svc, nodeId, ver, head, n.onConnected, n.onDisconnected).connect()
	case nodeConnected:
		dialer := util.SplitSlc1[*nodeDialer](job.Data)
		set, _ := n.svcToDialer.GetOrNew(dialer.svc, func() *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer] {
			return ds.NewSet2Item[kiwi.TSvc, int64, kiwi.INodeDialer](dialer.svc, 2, func(dialer kiwi.INodeDialer) int64 {
				return dialer.NodeId()
			})
		})
		old := set.Set(dialer)
		if old != nil {
			kiwi.Error2(util.EcExist, util.M{
				"node id": dialer.nodeId,
			})
		}
		_ = n.idToDialer.Set(dialer)
		kiwi.Info("service connected", util.M{
			"svc":     dialer.svc,
			"ver":     dialer.ver,
			"node id": dialer.nodeId,
			"head":    dialer.head,
		})
		//发送消息监听
		codes, ok := kiwi.Router().GetWatchCodes(dialer.Svc())
		if ok {
			bytes := kiwi.Packer().PackWatchNotify(kiwi.GetNodeMeta().NodeId, codes)
			dialer.Send(bytes, kiwi.Error)
		}
		var head util.M
		dialer.head.CopyTo(head)
		kiwi.DispatchEvent(kiwi.Evt_Svc_Connected, &kiwi.EvtSvcConnected{
			Svc:  dialer.svc,
			Id:   dialer.nodeId,
			Head: head,
		})
	case nodeDisconnected:
		dialer, err := util.SplitSlc2[*nodeDialer, *util.Err](job.Data)
		set, ok := n.svcToDialer.Get(dialer.svc)
		if !ok {
			return
		}
		nodeId := dialer.nodeId
		_, ok = set.Del(nodeId)
		if !ok {
			return
		}
		_, _ = n.idToDialer.Del(nodeId)
		codes, ok := n.watcherToCodes[nodeId]
		if ok {
			for _, code := range codes {
				m, ok := n.codeToWatchers[code]
				if ok {
					delete(m, nodeId)
				}
			}
		}
		kiwi.Info("dialer disconnected", util.M{
			"error":   err,
			"svc":     dialer.svc,
			"node id": dialer.nodeId,
			"head":    dialer.head,
		})
		kiwi.DispatchEvent(kiwi.Evt_Svc_Disonnected, &kiwi.EvtSvcDisconnected{
			Svc: dialer.svc,
			Id:  dialer.nodeId,
		})
	case nodeDisconnect:
		svc, nodeId := util.SplitSlc2[kiwi.TSvc, int64](job.Data)
		set, ok := n.svcToDialer.Get(svc)
		if !ok {
			return
		}
		dialer, ok := set.Get(nodeId)
		if !ok {
			return
		}
		kiwi.Info("disconnect service", util.M{
			"service": dialer.Svc(),
			"node id": dialer.NodeId(),
			"head":    dialer.Head(),
		})
		dialer.Dialer().Agent().Dispose()
	case nodeSendNotify:
		ntf := job.Data[0].(kiwi.ISndNotice)
		tid := ntf.Tid()
		bytes, err := kiwi.Packer().PackNotify(tid, ntf)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		m, ok := n.codeToWatchers[ntf.Code()]
		if !ok {
			return
		}
		for nodeId := range m {
			dialer, ok := n.idToDialer.Get(nodeId)
			if !ok {
				delete(m, nodeId)
				break
			}
			dialer.Send(util.CopyBytes(bytes), nil)
		}
	case nodeWatchNotify:
		nodeId, codes := util.SplitSlc2[int64, []kiwi.TCode](job.Data)
		_, ok := n.idToDialer.Get(nodeId)
		if !ok {
			kiwi.Error2(util.EcNotExist, util.M{
				"node id": nodeId,
			})
			return
		}
		n.watcherToCodes[nodeId] = codes
		for _, code := range codes {
			m, ok := n.codeToWatchers[code]
			if ok {
				m[nodeId] = struct{}{}
			} else {
				n.codeToWatchers[code] = map[int64]struct{}{
					nodeId: {},
				}
			}
		}
	case nodePush:
		pus := job.Data[0].(kiwi.ISndPush)
		tid := pus.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, pus)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToSvc(pus.Svc(), bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodePushNode:
		nodeId, pus := util.SplitSlc2[int64, kiwi.ISndPush](job.Data)
		tid := pus.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, pus)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(nodeId, bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeRequest:
		req := job.Data[0].(kiwi.ISndRequest)
		tid := req.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, req)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToSvc(req.Svc(), bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeRequestNode:
		nodeId, req := util.SplitSlc2[int64, kiwi.ISndRequest](job.Data)
		tid := req.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, req)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(nodeId, bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeSendNode:
		n.sendToNode(util.SplitSlc3[int64, []byte, util.FnErr](job.Data))
	}
}

func (n *nodeNet) sendToSvc(svc kiwi.TSvc, bytes []byte, fnErr util.FnErr) {
	set, ok := n.svcToDialer.Get(svc)
	if !ok {
		fnErr(util.NewErr(util.EcNotExist, util.M{
			"svc": svc,
		}))
		return
	}
	switch set.Count() {
	case 0:
		fnErr(util.NewErr(util.EcNotExist, util.M{
			"svc": svc,
		}))
	case 1:
		dialer, _ := set.GetWithIdx(0)
		dialer.Send(bytes, fnErr)
	default:
		nodeId, err := n.option.selector(set)
		if err != nil {
			fnErr(err)
			return
		}
		dialer, ok := n.idToDialer.Get(nodeId)
		if !ok {
			fnErr(util.NewErr(util.EcNotExist, util.M{
				"id": nodeId,
			}))
			return
		}
		dialer.Send(bytes, fnErr)
	}
}

func (n *nodeNet) sendToNode(nodeId int64, bytes []byte, fnErr util.FnErr) {
	dialer, ok := n.idToDialer.Get(nodeId)
	if !ok {
		fnErr(util.NewErr(util.EcNotExist, util.M{
			"id": nodeId,
		}))
		return
	}
	dialer.Send(bytes, fnErr)
}

const (
	nodeConnect      = "svc_connect"
	nodeConnected    = "svc_connected"
	nodeDisconnect   = "svc_disconnect"
	nodeDisconnected = "svc_disconnected"
	nodeSendNotify   = "send_notify"
	nodeWatchNotify  = "listen_notify"
	nodePush         = "push"
	nodePushNode     = "push_node"
	nodeRequest      = "request"
	nodeRequestNode  = "request_node"
	nodeSendNode     = "send_node"
)
