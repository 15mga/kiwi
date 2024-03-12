package ds

import "github.com/15mga/kiwi/util"

func NewLink2[T comparable]() *Link2[T] {
	return &Link2[T]{
		def: util.Default[T](),
	}
}

type Link2[T comparable] struct {
	head  *link2Elem[T]
	count uint32
	def   T
}

func (l *Link2[T]) Count() uint32 {
	return l.count
}

func (l *Link2[T]) Push(item T) {
	if l.count == 0 {
		e := newLink2Elem[T]()
		l.head = e
	}
	_ = l.head.push(item)
	l.count++
}

func (l *Link2[T]) Del(item T) bool {
	if l.head == nil {
		return false
	}
	ok := l.head.del(item, l.def)
	if ok {
		l.count--
	}
	return ok
}

func (l *Link2[T]) Iter(fn func([]T)) {
	if l.head == nil {
		return
	}
	l.head.iter(fn)
}

func (l *Link2[T]) Reset() {
	if l.head == nil {
		return
	}
	l.head.reset(l.def)
	l.head = nil
	l.count = 0
}

func (l *Link2[T]) IterAndReset(fn func([]T)) {
	if l.head == nil {
		return
	}
	l.head.iterAndReset(fn, l.def)
	l.head = nil
}

func newLink2Elem[T comparable]() *link2Elem[T] {
	return &link2Elem[T]{
		slc: make([]T, 32),
		cap: 32,
	}
}

type link2Elem[T comparable] struct {
	slc  []T
	idx  uint8
	cap  uint8
	next *link2Elem[T]
}

func (e *link2Elem[T]) push(item T) *link2Elem[T] {
	if e.idx < e.cap {
		e.slc[e.idx] = item
		e.idx++
		return e
	}
	ne := newLink2Elem[T]()
	ne.push(item)
	return ne
}

func (e *link2Elem[T]) del(item T, def T) bool {
	for i, t := range e.slc {
		if t == item {
			e.idx--
			if uint8(i) == e.idx {
				e.slc[i] = def
				return true
			}
			e.slc[i] = e.slc[e.idx]
			e.slc[e.idx] = def
			return true
		}
	}
	if e.next != nil {
		return e.next.del(item, def)
	}
	return false
}

func (e *link2Elem[T]) iter(fn func([]T)) {
	fn(e.slc[:e.idx])
	if e.next != nil {
		e.next.iter(fn)
	}
}

func (e *link2Elem[T]) reset(def T) {
	for i := uint8(0); i < e.idx; i++ {
		e.slc[i] = def
	}
	e.idx = 0
	if e.next != nil {
		e.next.reset(def)
		e.next = nil
	}
}

func (e *link2Elem[T]) iterAndReset(fn func([]T), def T) {
	fn(e.slc[:e.idx])
	for i := uint8(0); i < e.idx; i++ {
		e.slc[i] = def
	}
	e.idx = 0
	if e.next != nil {
		e.next.iterAndReset(fn, def)
		e.next = nil
	}
}
