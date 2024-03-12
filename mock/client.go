package mock

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/graph"
	"github.com/15mga/kiwi/graph/marshall"
	"github.com/15mga/kiwi/network"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
)

type (
	MsgToStrAny func(client *Client, msg util.IMsg) (point string, data any)
	Decoder     func(kiwi.IAgent, []byte) (svc kiwi.TSvc, mtd kiwi.TCode, msg util.IMsg, err *util.Err)
)

type Receiver struct {
	Node string
	Fn   MsgToStrAny
}

type Option struct {
	Id      string
	Name    string
	Ip      string
	Port    int
	Mermaid []byte
	Decoder Decoder
	Head    util.M
	HeadLen uint32
}

func NewClient(opt Option) (*Client, *util.Err) {
	g := graph.NewGraph("client")
	var ug marshall.Graph
	err := ug.Unmarshall(opt.Mermaid, g)
	if err != nil {
		return nil, err
	}
	m := opt.Head
	if m == nil {
		m = util.M{}
	}
	g.SetData(m)

	addr := fmt.Sprintf("%s:%d", opt.Ip, opt.Port)
	client := &Client{
		id:            opt.Id,
		decoder:       opt.Decoder,
		graph:         g,
		msgToReceiver: make(map[kiwi.TSvcCode]*Receiver),
	}
	m.Set("client", client)
	dialer := network.NewTcpDialer(opt.Name, addr, client.Receive,
		kiwi.AgentHeadLen(opt.HeadLen),
	)
	dialer.Agent().BindDisconnected(func(agent kiwi.IAgent, err *util.Err) {
		kiwi.Info("disconnect", util.M{
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
	return client, nil
}

type Client struct {
	id            string
	decoder       Decoder
	dialer        kiwi.IDialer
	graph         graph.IGraph
	msgToReceiver map[kiwi.TSvcCode]*Receiver
}

func (c *Client) Id() string {
	return c.id
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
	point, data := receiver.Fn(c, pkt)
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
	worker.Share().Push(c.id, c.jobOut, nd, point, data)
}

func (c *Client) jobOut(params []any) {
	nd, point, data := util.SplitSlc3[graph.INode, string, any](params)
	kiwi.Error(nd.Out(point, data))
}

func (c *Client) Push(fn util.FnAnySlc, params ...any) {
	worker.Share().Push(c.id, fn, params...)
}

func (c *Client) BindPointMsg(node, inPoint string, fn graph.MsgToErr) {
	nd, err := c.graph.GetNode(node)
	if err != nil {
		kiwi.Error(err)
		return
	}
	nd.BindFn(inPoint, func(msg graph.IMsg) *util.Err {
		kiwi.Info("process:", msg.ToM())
		return fn(msg)
	})
}

func (c *Client) BindNetMsg(node string, msg util.IMsg, receiver MsgToStrAny) {
	svc, mtd := kiwi.Codec().MsgToSvcCode(msg)
	c.msgToReceiver[kiwi.MergeSvcCode(svc, mtd)] = &Receiver{
		Node: node,
		Fn:   receiver,
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
