package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"strconv"
	"time"
)

const (
	HeadId    = "id"
	HeadSvc   = "svc"
	HeadCode  = "cod"
	HeadSndId = "snd_id"
	HeadSndTs = "snd_ts"
)

type sndPkt struct {
	pid     int64
	tid     int64
	sndTs   int64
	json    bool
	svc     kiwi.TSvc
	code    kiwi.TCode
	head    util.M
	payload []byte
	msg     util.IMsg
}

func (p *sndPkt) InitHead() {
	p.head.Set(HeadSvc, p.svc)
	p.head.Set(HeadCode, p.code)
	ni := kiwi.GetNodeMeta()
	p.head.Set(HeadSndId, ni.NodeId)
	p.head.Set(HeadSndTs, time.Now().UnixMilli())
}

func (p *sndPkt) Pid() int64 {
	return p.pid
}

func (p *sndPkt) Tid() int64 {
	return p.tid
}

func (p *sndPkt) Json() bool {
	return p.json
}

func (p *sndPkt) Svc() kiwi.TSvc {
	return p.svc
}

func (p *sndPkt) Code() kiwi.TCode {
	return p.code
}

func (p *sndPkt) Head() util.M {
	return p.head
}

func (p *sndPkt) GetSvcNodeId() (int64, bool) {
	id, ok := util.MGet[int64](p.head, strconv.Itoa(int(p.svc)))
	return id, ok
}

func (p *sndPkt) Payload() []byte {
	return p.payload
}

func (p *sndPkt) Msg() util.IMsg {
	return p.msg
}

func (p *sndPkt) Dispose() {
	p.msg = nil
}
