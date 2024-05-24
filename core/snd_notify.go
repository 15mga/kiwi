package core

import (
	"sync"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

var (
	_NtfPool = sync.Pool{
		New: func() any {
			return &SNotify{}
		},
	}
)

func newNotify(pid int64, head util.M, msg util.IMsg) *SNotify {
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
	ntf := _NtfPool.Get().(*SNotify)
	ntf.pid = pid
	ntf.tid = kiwi.TC(pid, head, IsExcludeLog(svc, code))
	ntf.svc, ntf.code = svc, code
	ntf.json = false
	ntf.head = head
	ntf.payload = payload
	ntf.InitHead()
	return ntf
}

type SNotify struct {
	sndPkt
}

func (n *SNotify) Dispose() {
	n.sndPkt.Dispose()
	_NtfPool.Put(n)
}

func Ntf(pid int64, head util.M, msg util.IMsg) int64 {
	ntf := newNotify(pid, head, msg)
	kiwi.Node().Notify(ntf, nil)
	return ntf.tid
}

func NtfWithFilter(pid int64, head util.M, msg util.IMsg, filter util.MToBool) int64 {
	ntf := newNotify(pid, head, msg)
	kiwi.Node().Notify(ntf, filter)
	return ntf.tid
}

func NtfOne(pid int64, head util.M, msg util.IMsg, filter util.MToBool) int64 {
	ntf := newNotify(pid, head, msg)
	kiwi.Node().NotifyOne(ntf, filter)
	return ntf.tid
}
