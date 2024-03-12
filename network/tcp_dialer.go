package network

import (
	"context"
	"github.com/15mga/kiwi"
	"net"

	"github.com/15mga/kiwi/util"
)

type tcpDialer struct {
	name  string
	agent *tcpAgent
}

func NewTcpDialer(name, addr string, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) kiwi.IDialer {
	d := &tcpDialer{
		name:  name,
		agent: NewTcpAgent(addr, receiver, options...),
	}
	return d
}

func (d *tcpDialer) Name() string {
	return d.name
}

func (d *tcpDialer) Connect(ctx context.Context) *util.Err {
	addr := d.agent.Addr()
	ta, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return util.NewErr(util.EcConnectErr, util.M{
			"addr":  addr,
			"error": err.Error(),
		})
	}

	c, err := net.DialTCP("tcp", nil, ta)
	if err != nil {
		return util.NewErr(util.EcConnectErr, util.M{
			"addr":  addr,
			"error": err.Error(),
		})
	}
	_ = c.SetNoDelay(true)
	d.agent.Start(ctx, c)
	return nil
}

func (d *tcpDialer) Agent() kiwi.IAgent {
	return d.agent
}
