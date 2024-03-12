package network

import (
	"context"
	"github.com/15mga/kiwi"

	"github.com/15mga/kiwi/util"
	"github.com/xtaci/kcp-go/v5"
)

type udpDialer struct {
	name  string
	agent kiwi.IAgent
}

func NewUdpDialer(name, addr string, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) kiwi.IDialer {
	d := &udpDialer{
		name:  name,
		agent: NewUdpAgent(addr, receiver, options...),
	}
	return d
}

func (d *udpDialer) Name() string {
	return d.name
}

func (d *udpDialer) Connect(ctx context.Context) *util.Err {
	addr := d.agent.Addr()
	c, err := kcp.Dial(addr)
	if err != nil {
		return util.NewErr(util.EcConnectErr, util.M{
			"addr":  addr,
			"error": err.Error(),
		})
	}

	d.agent.(*udpAgent).Start(ctx, c)
	return nil
}

func (d *udpDialer) Agent() kiwi.IAgent {
	return d.agent
}
