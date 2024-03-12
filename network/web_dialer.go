package network

import (
	"context"
	"github.com/15mga/kiwi"
	"net/http"

	"github.com/15mga/kiwi/util"
	"github.com/fasthttp/websocket"
)

type websocketDialer struct {
	name     string
	url      string
	header   http.Header
	agent    *webAgent
	receiver kiwi.FnAgentBytes
}

func NewWebDialer(url string, header http.Header, msgType int, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) kiwi.IDialer {
	d := &websocketDialer{
		url:      url,
		header:   header,
		receiver: receiver,
		agent:    NewWebAgent(url, msgType, receiver, options...),
	}
	return d
}

func (d *websocketDialer) Name() string {
	return d.name
}

func (d *websocketDialer) Connect(ctx context.Context) *util.Err {
	conn, _, err := websocket.DefaultDialer.Dial(d.url, d.header)
	if err != nil {
		return util.NewErr(util.EcConnectErr, util.M{
			"url":   d.url,
			"error": err.Error(),
		})
	}

	d.agent.Start(ctx, conn)
	return nil
}

func (d *websocketDialer) Agent() kiwi.IAgent {
	return d.agent
}
