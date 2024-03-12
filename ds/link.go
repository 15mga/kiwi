package ds

import (
	"github.com/15mga/kiwi/util"
)

func NewLink[T any]() *Link[T] {
	return &Link[T]{}
}

type Link[T any] struct {
	head  *LinkElem[T]
	tail  *LinkElem[T]
	count uint32
}

func (l *Link[T]) Count() uint32 {
	return l.count
}

func (l *Link[T]) Head() (T, bool) {
	if l.count == 0 {
		return util.Default[T](), false
	}
	return l.head.Value, true
}

func (l *Link[T]) Tail() (T, bool) {
	if l.tail == nil {
		return util.Default[T](), false
	}
	return l.tail.Value, true
}

func (l *Link[T]) Push(a T) {
	e := &LinkElem[T]{
		Value: a,
	}
	if l.count == 0 {
		l.head = e
	} else {
		l.tail.Next = e
	}
	l.tail = e
	l.count++
}

func (l *Link[T]) Pop() (T, bool) {
	if l.count == 0 {
		return util.Default[T](), false
	}
	e := l.head.Value
	l.head = l.head.Next
	l.count--
	if l.count == 0 {
		l.tail = nil
	}
	return e, true
}

func (l *Link[T]) Insert(a T, fn func(T) bool) {
	ne := &LinkElem[T]{
		Value: a,
	}
	l.count++
	if l.count == 0 {
		l.head = ne
		l.tail = ne
		return
	}
	var (
		pe, ce *LinkElem[T]
	)
	for ce = l.head; ce != nil; ce = ce.Next {
		if fn(ce.Value) {
			break
		}
		pe = ce
	}
	switch {
	case ce == l.head:
		ne.Next = ce
		l.head = ne
	case ce == nil:
		l.tail.Next = ne
		l.tail = ne
	default:
		pe.Next = ne
		ne.Next = ce
	}
}

func (l *Link[T]) Iter(fn func(T)) {
	for e := l.head; e != nil; e = e.Next {
		fn(e.Value)
	}
}

func (l *Link[T]) Any(fn func(T) bool) bool {
	for e := l.head; e != nil; e = e.Next {
		if fn(e.Value) {
			return true
		}
	}
	return false
}

func (l *Link[T]) Del(fn func(T) bool) bool {
	if l.count == 0 {
		return false
	}
	if fn(l.head.Value) {
		l.head = l.head.Next
		return true
	}
	p := l.head
	for p.Next != nil {
		if fn(p.Next.Value) {
			p.Next = p.Next.Next
			return true
		}
	}
	return false
}

func (l *Link[T]) Values(values *[]T) bool {
	if l.count == 0 {
		return false
	}
	l.Iter(func(v T) {
		*values = append(*values, v)
	})
	return true
}

func (l *Link[T]) Dispose() {
	l.head = nil
	l.tail = nil
	l.count = 0
}

func (l *Link[T]) PopAll() *LinkElem[T] {
	head := l.head
	l.head = nil
	l.tail = nil
	l.count = 0
	return head
}

func (l *Link[T]) Copy(lnk *Link[T]) {
	lnk.head = l.head
	lnk.tail = l.tail
	lnk.count = l.count
}

func (l *Link[T]) PushLink(lnk *Link[T]) {
	if lnk.count == 0 {
		return
	}
	if l.tail == nil {
		l.head = lnk.head
		l.tail = lnk.tail
		l.count = lnk.count
		return
	}
	l.tail.Next = lnk.head
	l.tail = lnk.tail
	l.count += lnk.count
}

type LinkElem[T any] struct {
	Next  *LinkElem[T]
	Value T
}
