package ds

import (
	"github.com/15mga/kiwi/util"
)

type (
	ringOption[T any] struct {
		maxCap      int
		minCap      int
		shrinkCount int
		slow        int
		resize      func(int)
	}
	RingOption[T any] func(o *ringOption[T])
)

func RingMaxCap[T any](c int) RingOption[T] {
	return func(o *ringOption[T]) {
		o.maxCap = c
	}
}

func RingMinCap[T any](c int) RingOption[T] {
	return func(o *ringOption[T]) {
		o.minCap = c
	}
}

func RingResize[T any](r func(int)) RingOption[T] {
	return func(o *ringOption[T]) {
		o.resize = r
	}
}

func RingSlow[T any](r func(int)) RingOption[T] {
	return func(o *ringOption[T]) {
		o.resize = r
	}
}

func NewRing[T any](opts ...RingOption[T]) *Ring[T] {
	opt := &ringOption[T]{
		maxCap:      0,
		minCap:      32,
		shrinkCount: 64,
		slow:        1024,
	}
	for _, o := range opts {
		o(opt)
	}
	r := &Ring[T]{
		opt:         opt,
		buffer:      make([]T, opt.minCap),
		bufferCap:   opt.minCap,
		halfBuffCap: opt.minCap >> 1,
		shrink:      opt.shrinkCount,
	}
	r.defVal = r.buffer[0]
	return r
}

type Ring[T any] struct {
	opt         *ringOption[T]
	defVal      T
	available   int
	readIdx     int
	writeIdx    int
	buffer      []T
	bufferCap   int
	halfBuffCap int
	shrink      int
}

func (r *Ring[T]) Available() int {
	return r.available
}

func (r *Ring[T]) testCap(c int) *util.Err {
	if c > r.bufferCap {
		c, ok := util.NextCap(c, r.bufferCap, r.opt.slow)
		if ok {
			if r.opt.maxCap > 0 && c >= r.opt.maxCap {
				return util.NewErr(util.EcTooLong, util.M{
					"total": c,
				})
			}
			r.resetBuffer(c)
		}
		return nil
	}
	if r.opt.minCap == r.bufferCap {
		return nil
	}
	if c > r.halfBuffCap {
		r.shrink = r.opt.shrinkCount
		return nil
	}
	r.shrink--
	if r.shrink > 0 {
		return nil
	}
	r.resetBuffer(r.halfBuffCap)
	return nil
}

func (r *Ring[T]) resetBuffer(cap int) {
	buf := make([]T, cap)
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
	r.shrink = r.opt.shrinkCount
	if r.opt.resize != nil {
		r.opt.resize(cap)
	}
}

func (r *Ring[T]) Put(items ...T) *util.Err {
	l := len(items)
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

func (r *Ring[T]) Pop() (item T, err *util.Err) {
	if r.available == 0 {
		return util.Default[T](), util.NewErr(util.EcNotEnough, util.M{
			"available": r.available,
		})
	}
	item = r.buffer[r.readIdx]
	r.readIdx++
	if r.readIdx == r.bufferCap {
		r.readIdx = 0
	}
	r.available--
	return
}

func (r *Ring[T]) Read(s []T, l int) *util.Err {
	sl := len(s)
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

func (r *Ring[T]) read(s []T, l int) {
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

func (r *Ring[T]) ReadMax(s []T) int {
	l := util.MaxInt(len(s), r.available)
	r.read(s, l)
	return l
}

func (r *Ring[T]) IterAll(fn func(T)) {
	if r.readIdx < r.writeIdx {
		for ; r.readIdx < r.writeIdx; r.readIdx++ {
			fn(r.buffer[r.readIdx])
		}
		return
	}
	for ; r.readIdx < r.bufferCap; r.readIdx++ {
		fn(r.buffer[r.readIdx])
	}
	for r.readIdx = 0; r.readIdx < r.writeIdx; r.readIdx++ {
		fn(r.buffer[r.readIdx])
	}
}

func (r *Ring[T]) Iter(fn func([]T)) {
	if r.readIdx == r.writeIdx {
		return
	}
	if r.readIdx < r.writeIdx {
		fn(r.buffer[r.readIdx:r.writeIdx])
		return
	}
	fn(r.buffer[r.readIdx:])
	fn(r.buffer[:r.writeIdx])
}

func (r *Ring[T]) Reset() {
	r.readIdx = 0
	r.writeIdx = 0
}
