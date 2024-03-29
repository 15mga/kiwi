package worker

import (
	"github.com/15mga/kiwi/util"
	"unsafe"
)

var (
	_Share *fnShare
)

func Share() *fnShare {
	return _Share
}

func InitShare() {
	if _Share != nil {
		return
	}
	_Share = &fnShare{
		count:   int64(_ParallelNum),
		workers: make([]*FnWorker, _ParallelNum),
	}
	_Share.mask = _Share.count - 1
	for i := 0; i < _ParallelNum; i++ {
		w := NewFnWorker()
		w.Start()
		_Share.workers[i] = w
	}
}

type fnShare struct {
	workers []*FnWorker
	count   int64
	mask    int64
	c       int64
}

func (s *fnShare) Push(key string, fn util.FnAnySlc, params ...any) {
	s.workers[FnvStr(key)&s.mask].Push(fn, params...)
}

func (s *fnShare) Dispose() {
	for _, worker := range s.workers {
		worker.Dispose()
	}
}

var (
	offset64 int64 = -3750763034362895579
	prime64  int64 = 1099511628211
)

func FnvStr(key string) int64 {
	bytes := util.StrToBytes(key)
	var hash = offset64
	for i := 0; i < len(bytes); i++ {
		hash ^= int64(bytes[i])
		hash *= prime64
	}
	return hash
}

func FnvBytes(bytes []byte) int64 {
	var hash = offset64
	for i := 0; i < len(bytes); i++ {
		hash ^= int64(bytes[i])
		hash *= prime64
	}
	return hash
}

func FnvInt64(v int64) int64 {
	var hash = offset64
	bytes := util.Int64ToBytes(v)
	for i := 0; i < len(bytes); i++ {
		hash ^= int64(bytes[i])
		hash *= prime64
	}
	return hash
}

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

type stringStruct struct {
	str unsafe.Pointer
	len int
}

func MemHash(data []byte) int64 {
	ss := (*stringStruct)(unsafe.Pointer(&data))
	return int64(memhash(ss.str, 0, uintptr(ss.len)))
}

func MemHashString(str string) int64 {
	ss := (*stringStruct)(unsafe.Pointer(&str))
	return int64(memhash(ss.str, 0, uintptr(ss.len)))
}
