package network

import (
	"context"
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"net"
	"time"

	"github.com/15mga/kiwi/util"
)

func NewUdpAgent(addr string, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) *udpAgent {
	return &udpAgent{
		agent: newAgent(addr, receiver, options...),
	}
}

type udpAgent struct {
	agent
	conn net.Conn
}

func (a *udpAgent) Start(ctx context.Context, conn net.Conn) {
	a.conn = conn
	a.onClose = a.conn.Close
	a.start(ctx)
	switch a.option.AgentMode {
	case kiwi.AgentRW:
		go a.read()
		go a.write()
	case kiwi.AgentR:
		go a.read()
	case kiwi.AgentW:
		go a.write()
	}
}

func (a *udpAgent) read() {
	var (
		newData [2048]byte
		err     *util.Err
	)
	defer func() {
		r := recover()
		if r != nil {
			kiwi.Error2(util.EcRecover, util.M{
				"remote addr": a.conn.RemoteAddr().String(),
				"recover":     fmt.Sprintf("%s", r),
			})
			a.read()
			return
		}
		a.close(err)
	}()

	dur := time.Duration(a.option.DeadlineSecs)
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if dur > 0 {
				_ = a.conn.SetReadDeadline(time.Now().Add(time.Second * dur))
			}
			newLen, e := a.conn.Read(newData[:])
			if e != nil {
				err = util.WrapErr(util.EcIo, e)
				return
			}
			a.receiver(a, newData[:newLen])
		}
	}
}

func (a *udpAgent) write() {
	var (
		err *util.Err
	)
	defer func() {
		a.close(err)
	}()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.writeSignCh:
			var elem *ds.LinkElem[[]byte]
			a.enable.Mtx.Lock()
			if a.enable.Disabled() {
				a.enable.Mtx.Unlock()
				return
			}
			elem = a.bytesLink.PopAll()
			a.enable.Mtx.Unlock()
			if elem == nil {
				continue
			}

			for ; elem != nil; elem = elem.Next {
				bytes := elem.Value
				_, e := a.conn.Write(bytes)
				util.RecycleBytes(bytes)
				if e != nil {
					err = util.WrapErr(util.EcIo, e)
					return
				}
			}
		}
	}
}
