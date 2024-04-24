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
	var ug marshall.Graph
	err := ug.Unmarshall(opt.Mermaid, g)
	if err != nil {
		return nil, err
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
	point, data := receiver.Fn(pkt)
	if point == "" {
		return
	}
	if data == nil {
		data = struct{}{}
	}
	nd, err := c.graph.GetNode(receiver.Node)
	if err != nil {
		kiwi.Error(err)
		return
	}
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

func (c *Client) BindPointMsg(node, inPoint string, fn graph.MsgToErr) {
	nd, err := c.graph.GetNode(node)
	if err != nil {
		return
	}
	nd.BindFn(inPoint, fn)
}

func (c *Client) BindNetMsg(msg util.IMsg, receiver MsgToStrAny) {
	svc, mtd := kiwi.Codec().MsgToSvcCode(msg)
	msgMethod := string(msg.ProtoReflect().Descriptor().Name())
	msgMethod, _ = strings.CutSuffix(msgMethod, "Res")
	msgMethod, _ = strings.CutSuffix(msgMethod, "Pus")
	var words []string
	util.SplitWords(msgMethod, &words)
	point := strings.ToLower(words[0])
	c.msgToReceiver[kiwi.MergeSvcCode(svc, mtd)] = &Receiver{
		Node:  point,
		Point: msgMethod,
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
