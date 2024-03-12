package ds

import (
	"reflect"

	"github.com/15mga/kiwi/util"
)

func NewFnErrLink() *FnErrLink {
	return &FnErrLink{
		Link: NewLink[func() *util.Err](),
	}
}

type FnErrLink struct {
	*Link[func() *util.Err]
}

func (l *FnErrLink) Invoke() *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink) Del(fn func() *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func() *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink1[T any]() *FnErrLink1[T] {
	return &FnErrLink1[T]{
		Link: NewLink[func(T) *util.Err](),
	}
}

type FnErrLink1[T any] struct {
	*Link[func(T) *util.Err]
}

func (l *FnErrLink1[T]) Invoke(obj T) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink1[T]) Del(fn func(T) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink1[T]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink2[T0, T1 any]() *FnErrLink2[T0, T1] {
	return &FnErrLink2[T0, T1]{
		Link: NewLink[func(T0, T1) *util.Err](),
	}
}

type FnErrLink2[T0, T1 any] struct {
	*Link[func(T0, T1) *util.Err]
}

func (l *FnErrLink2[T0, T1]) Invoke(v0 T0, v1 T1) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(v0, v1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink2[T0, T1]) Del(fn func(T0, T1) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink2[T0, T1]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink3[T0, T1, T2 any]() *FnErrLink3[T0, T1, T2] {
	return &FnErrLink3[T0, T1, T2]{
		Link: NewLink[func(T0, T1, T2) *util.Err](),
	}
}

type FnErrLink3[T0, T1, T2 any] struct {
	*Link[func(T0, T1, T2) *util.Err]
}

func (l *FnErrLink3[T0, T1, T2]) Invoke(v0 T0, v1 T1, v2 T2) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(v0, v1, v2)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink3[T0, T1, T2]) Del(fn func(T0, T1, T2) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink3[T0, T1, T2]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink4[T0, T1, T2, T3 any]() *FnErrLink4[T0, T1, T2, T3] {
	return &FnErrLink4[T0, T1, T2, T3]{
		Link: NewLink[func(T0, T1, T2, T3) *util.Err](),
	}
}

type FnErrLink4[T0, T1, T2, T3 any] struct {
	*Link[func(T0, T1, T2, T3) *util.Err]
}

func (l *FnErrLink4[T0, T1, T2, T3]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(v0, v1, v2, v3)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink4[T0, T1, T2, T3]) Del(fn func(T0, T1, T2, T3) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink4[T0, T1, T2, T3]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink5[T0, T1, T2, T3, T4 any]() *FnErrLink5[T0, T1, T2, T3, T4] {
	return &FnErrLink5[T0, T1, T2, T3, T4]{
		Link: NewLink[func(T0, T1, T2, T3, T4) *util.Err](),
	}
}

type FnErrLink5[T0, T1, T2, T3, T4 any] struct {
	*Link[func(T0, T1, T2, T3, T4) *util.Err]
}

func (l *FnErrLink5[T0, T1, T2, T3, T4]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3, v4 T4) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(v0, v1, v2, v3, v4)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink5[T0, T1, T2, T3, T4]) Del(fn func(T0, T1, T2, T3, T4) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3, T4) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink5[T0, T1, T2, T3, T4]) Reset() {
	l.head = nil
	l.tail = nil
}

func NewFnErrLink6[T0, T1, T2, T3, T4, T5 any]() *FnErrLink6[T0, T1, T2, T3, T4, T5] {
	return &FnErrLink6[T0, T1, T2, T3, T4, T5]{
		Link: NewLink[func(T0, T1, T2, T3, T4, T5) *util.Err](),
	}
}

type FnErrLink6[T0, T1, T2, T3, T4, T5 any] struct {
	*Link[func(T0, T1, T2, T3, T4, T5) *util.Err]
}

func (l *FnErrLink6[T0, T1, T2, T3, T4, T5]) Invoke(v0 T0, v1 T1, v2 T2, v3 T3, v4 T4, v5 T5) *util.Err {
	for e := l.head; e != nil; e = e.Next {
		err := e.Value(v0, v1, v2, v3, v4, v5)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *FnErrLink6[T0, T1, T2, T3, T4, T5]) Del(fn func(T0, T1, T2, T3, T4, T5) *util.Err) {
	pointer := reflect.ValueOf(fn).Pointer()
	_ = l.Link.Del(func(f func(T0, T1, T2, T3, T4, T5) *util.Err) bool {
		return reflect.ValueOf(f).Pointer() == pointer
	})
}

func (l *FnErrLink6[T0, T1, T2, T3, T4, T5]) Reset() {
	l.head = nil
	l.tail = nil
}
