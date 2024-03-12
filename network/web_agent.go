package network

import (
	"context"
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"time"

	"github.com/15mga/kiwi/util"
	"github.com/fasthttp/websocket"
)

func NewWebAgent(addr string, msgType int, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) *webAgent {
	return &webAgent{
		agent:   newAgent(addr, receiver, options...),
		msgType: msgType,
	}
}

type webAgent struct {
	agent
	msgType int
	conn    *websocket.Conn
}

func (a *webAgent) Start(ctx context.Context, conn *websocket.Conn) {
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

func (a *webAgent) read() {
	var err *util.Err
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
	c := a.conn
	c.SetReadLimit(int64(a.option.PacketMaxCap))
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if dur > 0 {
				_ = a.conn.SetReadDeadline(time.Now().Add(time.Second * dur))
			}
			mt, bytes, e := c.ReadMessage()
			if e != nil {
				err = util.WrapErr(util.EcIo, e)
				return
			}
			if mt != a.msgType {
				err = util.NewErr(util.EcWrongType, util.M{
					"receive message type": mt,
					"need message type":    a.msgType,
				})
				return
			}
			newLen := uint32(len(bytes))
			if newLen == 0 {
				break
			}
			a.receiver(a, bytes)
		}
	}
}

func (a *webAgent) write() {
	var (
		err *util.Err
	)
	defer func() {
		a.close(err)
	}()

	c := a.conn
	msgType := a.msgType
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
				e := c.WriteMessage(msgType, bytes)
				util.RecycleBytes(bytes)
				if e != nil {
					err = util.WrapErr(util.EcIo, e)
					return
				}
			}
		}
	}
}
