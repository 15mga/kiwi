package core

import (
	"context"
	"time"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

var (
	_NetReconnectDur = time.Second
	MaxReconnect     = 5
	MaxSendRetry     = uint8(3)
	SendRetryDur     = time.Second
)

type (
	OnNetDialerConnected    func(*nodeDialer)
	OnNetDialerDisconnected func(*nodeDialer, *util.Err)
)

func newNodeDialer(dialer kiwi.IDialer, svc kiwi.TSvc, nodeId int64, ver string,
	onConnected OnNetDialerConnected, onDisconnected OnNetDialerDisconnected) *nodeDialer {
	d := &nodeDialer{
		svc:            svc,
		nodeId:         nodeId,
		ver:            ver,
		dialer:         dialer,
		onConnected:    onConnected,
		onDisconnected: onDisconnected,
	}
	d.ctx, d.cancel = context.WithCancel(util.Ctx())
	dialer.Agent().BindConnected(d.onConn)
	dialer.Agent().BindDisconnected(d.onDisConn)
	return d
}

type nodeDialer struct {
	svc            kiwi.TSvc
	nodeId         int64
	ver            string
	dialer         kiwi.IDialer
	currReconnect  int
	onConnected    OnNetDialerConnected
	onDisconnected OnNetDialerDisconnected
	ctx            context.Context
	cancel         context.CancelFunc
}

func (d *nodeDialer) heartbeat() {
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-d.ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				d.Send(0, Heartbeat)
			}
		}
	}()
}

func (d *nodeDialer) onConn(_ kiwi.IAgent) {
	d.onConnected(d)
	if kiwi.GetNodeMeta().Mode == kiwi.ModeDebug {
		return
	}
	d.heartbeat()
}

func (d *nodeDialer) onDisConn(_ kiwi.IAgent, err *util.Err) {
	d.onDisconnected(d, err)
	d.cancel()
}

func (d *nodeDialer) Svc() kiwi.TSvc {
	return d.svc
}

func (d *nodeDialer) NodeId() int64 {
	return d.nodeId
}

func (d *nodeDialer) Dialer() kiwi.IDialer {
	return d.dialer
}

func (d *nodeDialer) connect() {
	err := d.dialer.Connect(util.Ctx())
	if err == nil {
		d.currReconnect = 0
		return
	}
	err.AddParams(util.M{
		"service":   d.svc,
		"node id":   d.nodeId,
		"reconnect": d.currReconnect,
	})
	kiwi.Error(err)
	d.reconnect()
}

func (d *nodeDialer) reconnect() {
	d.currReconnect++
	if d.currReconnect >= MaxReconnect {
		kiwi.Error2(util.EcConnectErr, util.M{
			"error":   "connect failed",
			"service": d.svc,
			"node id": d.nodeId,
		})
		return
	}
	time.Sleep(_NetReconnectDur)
	_ = d.dialer.Agent().Enable().IfEnable(d.connect)
}

func (d *nodeDialer) Send(tid int64, bytes []byte) {
	d.sendWithCount(tid, bytes, 0)
}

func (d *nodeDialer) sendWithCount(tid int64, bytes []byte, count uint8) {
	err := d.dialer.Agent().Send(bytes)
	if err == nil {
		return
	}
	if err.Code() != util.EcClosed {
		if tid > 0 {
			kiwi.TE(tid, err)
		} else {
			kiwi.Error(err)
		}
		return
	}
	if count >= MaxSendRetry {
		if tid > 0 {
			kiwi.TE(tid, err)
		} else {
			kiwi.Error(err)
		}
		return
	}
	time.AfterFunc(SendRetryDur, func() {
		d.sendWithCount(tid, bytes, count+1)
	})
}
