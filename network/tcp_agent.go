package network

import (
	"context"
	"fmt"
	"github.com/15mga/kiwi"
	"net"
	"time"

	"github.com/15mga/kiwi/ds"

	"github.com/15mga/kiwi/util"
)

// NewTcpAgent receiver接收字节如果使用异步方式需要copy一份，否则数据会被覆盖
func NewTcpAgent(addr string, receiver kiwi.FnAgentBytes, options ...kiwi.AgentOption) *tcpAgent {
	ta := &tcpAgent{
		agent: newAgent(addr, receiver, options...),
	}
	switch ta.option.HeadLen {
	case 2:
		ta.headReader = func(bytes []byte) uint32 {
			return uint32(bytes[0])<<8 | uint32(bytes[1])
		}
		ta.headWriter = func(buffer *util.ByteBuffer, bytes []byte) {
			buffer.WUint16(uint16(len(bytes)))
		}
	case 4:
		ta.headReader = func(bytes []byte) uint32 {
			return uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3])
		}
		ta.headWriter = func(buffer *util.ByteBuffer, bytes []byte) {
			buffer.WUint32(uint32(len(bytes)))
		}
	default:
		panic("wrong head length")
	}
	return ta
}

type tcpAgent struct {
	agent
	conn       net.Conn
	headReader util.BytesToUint32
	headWriter func(buffer *util.ByteBuffer, bytes []byte)
}

func (a *tcpAgent) Start(ctx context.Context, conn net.Conn) {
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

func (a *tcpAgent) read() {
	var (
		buffer     = make([]byte, a.option.PacketMinCap)
		ringBuffer = newRing(a.option.PacketMinCap, a.option.PacketMaxCap)
		pkgLen     uint32
		err        *util.Err
		headLen    = a.option.HeadLen
		headReader = a.headReader
		dur        = time.Duration(a.option.DeadlineSecs)
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

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if dur > 0 {
				_ = a.conn.SetReadDeadline(time.Now().Add(time.Second * dur))
			}
			newLen, e := a.conn.Read(buffer)
			if e != nil {
				err = util.WrapErr(util.EcIo, e)
				return
			}
			err = ringBuffer.Put(buffer[:newLen])
			if err != nil {
				return
			}
			for {
				if pkgLen == 0 {
					if ringBuffer.Available() < headLen {
						break
					}
					_ = ringBuffer.Read(buffer, headLen)
					pkgLen = headReader(buffer)
					if pkgLen == 0 {
						err = util.NewErr(util.EcBadHead, nil)
						return
					}
				}
				if ringBuffer.Available() < pkgLen {
					break
				}
				_ = ringBuffer.Read(buffer, pkgLen)
				//log.Debug("receive", util.M{
				//	"len": pkgLen,
				//	"hex": util.Hex(buffer[:pkgLen]),
				//})
				a.receiver(a, buffer[:pkgLen])
				pkgLen = 0
			}
		}
	}
}

func (a *tcpAgent) write() {
	var (
		err *util.Err
	)
	defer func() {
		a.close(err)
	}()

	headWriter := a.headWriter

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
				//log.Debug("send", util.M{
				//	"len": len(bytes),
				//	"hex": util.Hex(bytes),
				//})
				var buffer util.ByteBuffer
				buffer.InitCap(uint32(len(bytes)) + a.option.HeadLen)
				headWriter(&buffer, bytes)
				_, _ = buffer.Write(bytes)
				_, e := a.conn.Write(buffer.All())
				util.RecycleBytes(bytes)
				buffer.Dispose()
				if e != nil {
					err = util.WrapErr(util.EcIo, e)
					return
				}
			}
		}
	}
}

func newRing(minCap, maxCap uint32) *ring {
	r := &ring{
		buffer:      make([]byte, minCap),
		bufferCap:   minCap,
		halfBuffCap: minCap >> 1,
		minCap:      minCap,
		maxCap:      maxCap,
		shrink:      64,
		shrinkCount: 64,
	}
	r.defVal = r.buffer[0]
	return r
}

type ring struct {
	defVal      byte
	available   uint32
	readIdx     uint32
	writeIdx    uint32
	buffer      []byte
	bufferCap   uint32
	minCap      uint32
	maxCap      uint32
	halfBuffCap uint32
	shrink      uint32
	shrinkCount uint32
}

func (r *ring) Available() uint32 {
	return r.available
}

func (r *ring) testCap(c uint32) *util.Err {
	if c > r.bufferCap {
		c := util.NextPowerOfTwo(c)
		if r.maxCap > 0 && c >= r.maxCap {
			return util.NewErr(util.EcTooLong, util.M{
				"total": c,
			})
		}
		r.resetBuffer(c)
		return nil
	}
	if r.minCap == r.bufferCap {
		return nil
	}
	if c > r.halfBuffCap {
		r.shrink = r.shrinkCount
		return nil
	}
	r.shrink--
	if r.shrink > 0 {
		return nil
	}
	r.resetBuffer(r.halfBuffCap)
	return nil
}

func (r *ring) resetBuffer(cap uint32) {
	buf := make([]byte, cap)
	if r.available > 0 {
		if r.writeIdx > r.readIdx {
			copy(buf, r.buffer[r.readIdx:r.writeIdx])
		} else {
			n := copy(buf, r.buffer[r.readIdx:])
			copy(buf[n:], r.buffer[:r.writeIdx])
		}
	}
	r.writeIdx = r.available
	r.readIdx = 0
	r.bufferCap = cap
	r.halfBuffCap = cap >> 1
	r.buffer = buf
	r.shrink = r.shrinkCount
	r.buffer = make([]byte, cap)
}

func (r *ring) Put(items []byte) *util.Err {
	l := uint32(len(items))
	c := r.available + l
	err := r.testCap(c)
	if err != nil {
		return err
	}
	r.available = c
	i := r.writeIdx + l
	if i <= r.bufferCap {
		copy(r.buffer[r.writeIdx:], items)
		r.writeIdx = i
	} else {
		copy(r.buffer[r.writeIdx:r.bufferCap], items)
		j := r.bufferCap - r.writeIdx
		copy(r.buffer, items[j:l])
		r.writeIdx = l - j
	}
	return nil
}

func (r *ring) Read(s []byte, l uint32) *util.Err {
	sl := uint32(len(s))
	if l > sl || l > r.available {
		return util.NewErr(util.EcNotEnough, util.M{
			"length":    l,
			"slice":     sl,
			"available": r.available,
		})
	}
	r.read(s, l)
	return nil
}

func (r *ring) read(s []byte, l uint32) {
	p := r.readIdx + l
	if p < r.bufferCap {
		copy(s, r.buffer[r.readIdx:p])
		r.readIdx = p
	} else {
		p -= r.bufferCap
		copy(s, r.buffer[r.readIdx:r.bufferCap])
		copy(s[r.bufferCap-r.readIdx:], r.buffer[:p])
		r.readIdx = p
	}
	r.available -= l
}
