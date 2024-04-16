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
	n.worker = worker.NewWorker(512, n.processor)
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
	worker         *worker.Worker
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
	n.worker.Push(nodeJobConnect{ip, port, svc, nodeId, ver, head})
}

func (n *nodeNet) Disconnect(svc kiwi.TSvc, nodeId int64) {
	n.worker.Push(nodeJobDisconnect{svc, nodeId})
}

func (n *nodeNet) onConnected(dialer *nodeDialer) {
	n.worker.Push(nodeJobConnected{dialer})
}

func (n *nodeNet) onDisconnected(dialer *nodeDialer, err *util.Err) {
	n.worker.Push(nodeJobDisconnected{dialer, err})
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
	n.worker.Push(pus)
}

func (n *nodeNet) PushNode(nodeId int64, pus kiwi.ISndPush) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.pushSelf(pus)
		return
	}
	n.worker.Push(nodeJobPusNode{nodeId, pus})
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
	n.worker.Push(req)
}

func (n *nodeNet) RequestNode(nodeId int64, req kiwi.ISndRequest) {
	if nodeId == kiwi.GetNodeMeta().NodeId {
		n.requestSelf(req)
		return
	}
	n.worker.Push(nodeJobReqNode{nodeId, req})
}

func (n *nodeNet) Notify(ntf kiwi.ISndNotice) {
	n.worker.Push(ntf)
}

func (n *nodeNet) ReceiveWatchNotice(nodeId int64, codes []kiwi.TCode) {
	n.worker.Push(nodeJobWatchNotice{nodeId, codes})
}

func (n *nodeNet) SendToNode(nodeId int64, bytes []byte, fnErr util.FnErr) {
	n.worker.Push(nodeJobSendBytes{nodeId, bytes, fnErr})
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

func (n *nodeNet) processor(data any) {
	switch d := data.(type) {
	case nodeJobConnect:
		if n.idToDialer.Has(d.nodeId) {
			kiwi.Info("exist service", util.M{
				"node id": d.nodeId,
			})
			return
		}
		kiwi.Info("connect service", util.M{
			"ip":      d.ip,
			"port":    d.port,
			"svc":     d.svc,
			"node id": d.nodeId,
			"ver":     d.ver,
			"head":    d.head,
		})
		dialer := n.createDialer(fmt.Sprintf("%d_%d", d.svc, d.nodeId), fmt.Sprintf("%s:%d", d.ip, d.port))
		newNodeDialer(dialer, d.svc, d.nodeId, d.ver, d.head, n.onConnected, n.onDisconnected).connect()
	case nodeJobConnected:
		set, _ := n.svcToDialer.GetOrNew(d.dialer.svc, func() *ds.Set2Item[kiwi.TSvc, int64, kiwi.INodeDialer] {
			return ds.NewSet2Item[kiwi.TSvc, int64, kiwi.INodeDialer](d.dialer.svc, 2, func(dialer kiwi.INodeDialer) int64 {
				return dialer.NodeId()
			})
		})
		old := set.Set(d.dialer)
		if old != nil {
			kiwi.Error2(util.EcExist, util.M{
				"node id": d.dialer.nodeId,
			})
		}
		_ = n.idToDialer.Set(d.dialer)
		kiwi.Info("service connected", util.M{
			"svc":     d.dialer.svc,
			"ver":     d.dialer.ver,
			"node id": d.dialer.nodeId,
			"head":    d.dialer.head,
		})
		//发送消息监听
		codes, ok := kiwi.Router().GetWatchCodes(d.dialer.Svc())
		if ok {
			bytes := kiwi.Packer().PackWatchNotify(kiwi.GetNodeMeta().NodeId, codes)
			d.dialer.Send(bytes, kiwi.Error)
		}
		var head util.M
		d.dialer.head.CopyTo(head)
		kiwi.DispatchEvent(kiwi.Evt_Svc_Connected, &kiwi.EvtSvcConnected{
			Svc:  d.dialer.svc,
			Id:   d.dialer.nodeId,
			Head: head,
		})
	case nodeJobDisconnected:
		set, ok := n.svcToDialer.Get(d.dialer.svc)
		if !ok {
			return
		}
		nodeId := d.dialer.nodeId
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
			"error":   d.err,
			"svc":     d.dialer.svc,
			"node id": d.dialer.nodeId,
			"head":    d.dialer.head,
		})
		kiwi.DispatchEvent(kiwi.Evt_Svc_Disonnected, &kiwi.EvtSvcDisconnected{
			Svc: d.dialer.svc,
			Id:  d.dialer.nodeId,
		})
	case nodeJobDisconnect:
		set, ok := n.svcToDialer.Get(d.svc)
		if !ok {
			return
		}
		dialer, ok := set.Get(d.nodeId)
		if !ok {
			return
		}
		kiwi.Info("disconnect service", util.M{
			"service": dialer.Svc(),
			"node id": dialer.NodeId(),
			"head":    dialer.Head(),
		})
		dialer.Dialer().Agent().Dispose()
	case kiwi.ISndNotice:
		tid := d.Tid()
		bytes, err := kiwi.Packer().PackNotify(tid, d)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		m, ok := n.codeToWatchers[d.Code()]
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
	case nodeJobWatchNotice:
		_, ok := n.idToDialer.Get(d.nodeId)
		if !ok {
			kiwi.Error2(util.EcNotExist, util.M{
				"node id": d.nodeId,
			})
			return
		}
		n.watcherToCodes[d.nodeId] = d.codes
		for _, code := range d.codes {
			m, ok := n.codeToWatchers[code]
			if ok {
				m[d.nodeId] = struct{}{}
			} else {
				n.codeToWatchers[code] = map[int64]struct{}{
					d.nodeId: {},
				}
			}
		}
	case kiwi.ISndPush:
		tid := d.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, d)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToSvc(d.Svc(), bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeJobPusNode:
		tid := d.pus.Tid()
		bytes, err := kiwi.Packer().PackPush(tid, d.pus)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(d.nodeId, bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case kiwi.ISndRequest:
		tid := d.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, d)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToSvc(d.Svc(), bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeJobReqNode:
		tid := d.req.Tid()
		bytes, err := kiwi.Packer().PackRequest(tid, d.req)
		if err != nil {
			kiwi.TE(tid, err)
			return
		}
		n.sendToNode(d.nodeId, bytes, func(err *util.Err) {
			kiwi.TE(tid, err)
		})
	case nodeJobSendBytes:
		n.sendToNode(d.nodeId, d.payload, d.fnErr)
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

type nodeJobConnect struct {
	ip     string
	port   int
	svc    kiwi.TSvc
	nodeId int64
	ver    string
	head   util.M
}

type nodeJobConnected struct {
	dialer *nodeDialer
}

type nodeJobDisconnect struct {
	svc    kiwi.TSvc
	nodeId int64
}

type nodeJobDisconnected struct {
	dialer *nodeDialer
	err    *util.Err
}

type nodeJobWatchNotice struct {
	nodeId int64
	codes  []kiwi.TCode
}

type nodeJobPusNode struct {
	nodeId int64
	pus    kiwi.ISndPush
}

type nodeJobReqNode struct {
	nodeId int64
	req    kiwi.ISndRequest
}

type nodeJobSendBytes struct {
	nodeId  int64
	payload []byte
	fnErr   util.FnErr
}
