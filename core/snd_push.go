package core

import (
	"github.com/15mga/kiwi"
	"sync"

	"github.com/15mga/kiwi/util"
)

var (
	_PusPool = sync.Pool{
		New: func() any {
			return &SPush{}
		},
	}
	GenHead util.FnM = func(m util.M) {

	}
)

func newSPush(pid int64, head util.M, msg util.IMsg) *SPush {
	payload, err := kiwi.Codec().PbMarshal(msg)
	if err != nil {
		kiwi.Fatal(err)
		return nil
	}

	if head == nil {
		head = util.M{}
	}
	GenHead(head)
	svc, code := kiwi.Codec().MsgToSvcCode(msg)

	pus := _PusPool.Get().(*SPush)
	pus.msg = msg
	pus.pid = pid
	pus.svc, pus.code = svc, code
	pus.head = head
	pus.payload = payload
	pus.InitHead()
	pus.tid = kiwi.TC(pid, head, IsExcludeLog(svc, code))
	return pus
}

type SPush struct {
	sndPkt
}

func (p *SPush) Dispose() {
	p.sndPkt.Dispose()
	_PusPool.Put(p)
}

func Pus(pid int64, head util.M, msg util.IMsg) int64 {
	pus := newSPush(pid, head, msg)
	kiwi.Node().Push(pus)
	return pus.tid
}

func PusNode(pid, nodeId int64, head util.M, msg util.IMsg) int64 {
	pus := newSPush(pid, head, msg)
	kiwi.Node().PushNode(nodeId, pus)
	return pus.tid
}
