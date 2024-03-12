package ds

import (
	"reflect"
	"sync"

	"github.com/15mga/kiwi/util"
)

var (
	_FnLinkPool = sync.Pool{
		New: func() any {
			return &FnLink{
				Link: NewLink[util.Fn](),
			}
		},
	}
)

func NewFnLink() *FnLink {
	return _FnLinkPool.Get().(*FnLink)
}

type FnLink struct {
	*Link[util.Fn]
}

func (l *FnLink) Invoke() bool {
	if l.count == 0 {
		return false
	}
	for e := l.head; e != nil; e = e.Next {
		e.Value()
	}
	return true
}

func (l *FnLink) InvokeAndReset() bool {
	if l.count == 0 {
		return false
	}
	for e := l.head; e != nil; e = e.Next {
		e.Value()
	}
	l.head = nil
	l.tail = nil
	l.count = 0
	return true
}

func (l *FnLink) Dispose() {
	l.Link.Dispose()
	_FnLinkPool.Put(l)
}

func (l *FnLink) Del(fn util.Fn) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f util.Fn) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink) Reset() {
	if l.count == 0 {
		return
	}
	l.head = nil
	l.tail = nil
	l.count = 0
}

func NewFnLink1[T any]() *FnLink1[T] {
	l := &FnLink1[T]{
		Link: NewLink[func(T)](),
	}
	return l
}

type FnLink1[T any] struct {
	*Link[func(T)]
}

func (l *FnLink1[T]) Invoke(obj T) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(obj)
	}
}

func (l *FnLink1[T]) Del(fn func(T)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink1[T]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnLink2[T0, T1 any]() *FnLink2[T0, T1] {
	return &FnLink2[T0, T1]{
		Link: NewLink[func(T0, T1)](),
	}
}

type FnLink2[T0, T1 any] struct {
	*Link[func(T0, T1)]
}

func (l *FnLink2[T0, T1]) Invoke(v0 T0, v1 T1) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(v0, v1)
	}
}

func (l *FnLink2[T0, T1]) Del(fn func(T0, T1)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink2[T0, T1]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnLink3[T0, T1, T2 any]() *FnLink3[T0, T1, T2] {
	return &FnLink3[T0, T1, T2]{
		Link: NewLink[func(T0, T1, T2)](),
	}
}

type FnLink3[T0, T1, T2 any] struct {
	*Link[func(T0, T1, T2)]
}

func (l *FnLink3[T0, T1, T2]) Invoke(v0 T0, v1 T1, v2 T2) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(v0, v1, v2)
	}
}

func (l *FnLink3[T0, T1, T2]) Del(fn func(T0, T1, T2)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink3[T0, T1, T2]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnLink4[T0, T1, T2, T3 any]() *FnLink4[T0, T1, T2, T3] {
	return &FnLink4[T0, T1, T2, T3]{
		Link: NewLink[func(T0, T1, T2, T3)](),
	}
}

type FnLink4[T0, T1, T2, T3 any] struct {
	*Link[func(T0, T1, T2, T3)]
}

func (l *FnLink4[T0, T1, T2, T3]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(v0, v1, v2, v3)
	}
}

func (l *FnLink4[T0, T1, T2, T3]) Del(fn func(T0, T1, T2, T3)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink4[T0, T1, T2, T3]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnLink5[T0, T1, T2, T3, T4 any]() *FnLink5[T0, T1, T2, T3, T4] {
	return &FnLink5[T0, T1, T2, T3, T4]{
		Link: NewLink[func(T0, T1, T2, T3, T4)](),
	}
}

type FnLink5[T0, T1, T2, T3, T4 any] struct {
	*Link[func(T0, T1, T2, T3, T4)]
}

func (l *FnLink5[T0, T1, T2, T3, T4]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3, v4 T4) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(v0, v1, v2, v3, v4)
	}
}

func (l *FnLink5[T0, T1, T2, T3, T4]) Del(fn func(T0, T1, T2, T3, T4)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3, T4)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink5[T0, T1, T2, T3, T4]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnLink6[T0, T1, T2, T3, T4, T5 any]() *FnLink6[T0, T1, T2, T3, T4, T5] {
	return &FnLink6[T0, T1, T2, T3, T4, T5]{
		Link: NewLink[func(T0, T1, T2, T3, T4, T5)](),
	}
}

type FnLink6[T0, T1, T2, T3, T4, T5 any] struct {
	*Link[func(T0, T1, T2, T3, T4, T5)]
}

func (l *FnLink6[T0, T1, T2, T3, T4, T5]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3, v4 T4, v5 T5) {
	for e := l.head; e != nil; e = e.Next {
		e.Value(v0, v1, v2, v3, v4, v5)
	}
}

func (l *FnLink6[T0, T1, T2, T3, T4, T5]) Del(fn func(T0, T1, T2, T3, T4, T5)) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3, T4, T5)) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnLink6[T0, T1, T2, T3, T4, T5]) Reset() {
	l.head = nil
	l.tail = nil
}
