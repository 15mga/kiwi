package mock

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/graph"
	"github.com/15mga/kiwi/graph/marshall"
	"github.com/15mga/kiwi/network"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"strings"
)

type (
	MsgToStrAny func(msg util.IMsg) (point string, data any)
	Decoder     func(kiwi.IAgent, []byte) (svc kiwi.TSvc, mtd kiwi.TCode, msg util.IMsg, err *util.Err)
)

type Receiver struct {
	Node  string
	Point string
	Fn    MsgToStrAny
}

type Option struct {
	Name    string
	Ip      string
	Port    int
	Mermaid []byte
	Decoder Decoder
	Data    util.M
	HeadLen int
	Type    string
}

func NewClient(opt Option) (*Client, *util.Err) {
	g := graph.NewGraph("client")
	if opt.Mermaid != nil {
		var ug marshall.Graph
		err := ug.Unmarshall(opt.Mermaid, g)
		if err != nil {
			return nil, err
		}
	}
	m := opt.Data
	if m == nil {
		m = util.M{}
	}
	g.SetData(m)

	addr := fmt.Sprintf("%s:%d", opt.Ip, opt.Port)
	client := &Client{
		decoder:       opt.Decoder,
		graph:         g,
		msgToReceiver: make(map[kiwi.TSvcCode]*Receiver),
		worker:        worker.NewFnWorker(16),
	}
	m.Set("client", client)
	var dialer kiwi.IDialer
	switch opt.Type {
	case "web":
		dialer = network.NewWebDialer(addr, nil, 2, client.Receive)
	case "tcp":
		dialer = network.NewTcpDialer(opt.Name, addr, client.Receive,
			kiwi.AgentHeadLen(opt.HeadLen),
		)
	default:
		panic("unknown client type")
	}
	dialer.Agent().BindDisconnected(func(agent kiwi.IAgent, err *util.Err) {
		kiwi.Info("disconnected", util.M{
			"addr": agent.Addr(),
		})
	})
	dialer.Agent().BindConnected(func(agent kiwi.IAgent) {
		kiwi.Info("connected", util.M{
			"addr": agent.Addr(),
		})
	})
	e := dialer.Connect(util.Ctx())
	if e != nil {
		return nil, e
	}
	client.dialer = dialer
	client.worker.Start()
	return client, nil
}

type Client struct {
	decoder       Decoder
	dialer        kiwi.IDialer
	graph         graph.IGraph
	msgToReceiver map[kiwi.TSvcCode]*Receiver
	worker        *worker.FnWorker
}

func (c *Client) Dialer() kiwi.IDialer {
	return c.dialer
}

func (c *Client) Graph() graph.IGraph {
	return c.graph
}

func (c *Client) Receive(agent kiwi.IAgent, bytes []byte) {
	svc, mtd, pkt, err := c.decoder(agent, bytes)
	if err != nil {
		err.AddParam("payload", string(bytes[2:]))
		kiwi.Error(err)
		return
	}
	receiver, ok := c.msgToReceiver[kiwi.MergeSvcCode(svc, mtd)]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"svc":    svc,
			"method": mtd,
		})
		return
	}
	kiwi.Debug("receiver", util.M{
		string(pkt.ProtoReflect().Descriptor().Name()): pkt,
	})
	point, data := receiver.Fn(pkt)
	if point == "" {
		return
	}
	if data == nil {
		data = struct{}{}
	}
	nd := c.graph.GetNode(receiver.Node)
	c.worker.Push(c.jobOut, jobNodeOut{
		node:  nd,
		point: point,
		data:  data,
	})
}

type jobNodeOut struct {
	node  graph.INode
	point string
	data  any
}

func (c *Client) jobOut(params any) {
	job := params.(jobNodeOut)
	kiwi.Error(job.node.Out(job.point, job.data))
}

func (c *Client) Do(fn util.FnAny, params any) {
	c.worker.Push(fn, params)
}

func (c *Client) MsgToNodeAndPoint(msg util.IMsg) (node, point string) {
	point = string(msg.ProtoReflect().Descriptor().Name())
	point = point[:len(point)-3]
	//point, _ = strings.CutSuffix(point, "Res")
	//point, _ = strings.CutSuffix(point, "Req")
	//point, _ = strings.CutSuffix(point, "Pus")
	var words []string
	util.SplitWords(point, &words)
	node = strings.ToLower(words[0])
	return
}

func (c *Client) Link(out, in util.IMsg) (graph.ILink, *util.Err) {
	outNode, outPoint := c.MsgToNodeAndPoint(out)
	op := c.graph.GetNode(outNode)
	if !op.HasOut(outPoint) {
		_ = op.AddOut("nil", outPoint)
	}
	inNode, inPoint := c.MsgToNodeAndPoint(in)
	ip := c.graph.GetNode(inNode)
	if !ip.HasIn(inPoint) {
		_ = ip.AddIn("nil", inPoint)
	}
	return c.graph.Link(outNode, outPoint, inNode, inPoint)
}

func (c *Client) BindReqDecorator(msg util.IMsg, decorator func(*Client, util.IMsg)) {
	_, point := c.MsgToNodeAndPoint(msg)
	c.graph.Data().Set(point+"Decorator", decorator)
}

func (c *Client) DecorateReq(req util.IMsg) {
	_, point := c.MsgToNodeAndPoint(req)
	fn, ok := util.MGet[func(*Client, util.IMsg)](c.Graph().Data(), point+"Decorator")
	if ok && fn != nil {
		fn(c, req)
	}
}

//func (c *Client) Req(req util.IMsg) *util.Err {
//	kiwi.Debug("request", util.M{string(req.ProtoReflect().Descriptor().Name()): req})
//	svc, code := kiwi.Codec().MsgToSvcCode(req)
//	bytes, err := common.PackUserReq(svc, code, req)
//	if err != nil {
//		return err
//	}
//	return s.client.Dialer().Agent().Send(bytes)
//}

func (c *Client) BindPointMsg(node, inPoint string, fn graph.MsgToErr) {
	c.graph.GetNode(node).BindFn(inPoint, fn)
}

func (c *Client) BindNetMsg(msg util.IMsg, receiver MsgToStrAny) {
	node, point := c.MsgToNodeAndPoint(msg)
	svc, mtd := kiwi.Codec().MsgToSvcCode(msg)
	c.msgToReceiver[kiwi.MergeSvcCode(svc, mtd)] = &Receiver{
		Node:  node,
		Point: point,
		Fn:    receiver,
	}
}

func (c *Client) SetM(key string, data any) {
	c.graph.Data().Set(key, data)
}

func (c *Client) GetM(key string) (any, bool) {
	return c.graph.Data().Get(key)
}

func ClientGetM[T any](c *Client, key string) (T, bool) {
	return util.MGet[T](c.graph.Data(), key)
}

func GraphMsgGetM[T any](msg graph.IMsg, key string) (T, bool) {
	client, _ := util.MGet[*Client](msg.InNode().RootGraph().Data(), "client")
	return ClientGetM[T](client, key)
}

func MsgGetClient(msg graph.IMsg) *Client {
	client, _ := util.MGet[*Client](msg.InNode().RootGraph().Data(), "client")
	return client
}
