package kiwi

import (
	"reflect"

	"github.com/15mga/kiwi/ds"

	"github.com/15mga/kiwi/util"
)

var (
	_Dispatcher = NewDispatcher[util.M](util.M{})
)

func Event() *Dispatcher[util.M] {
	return _Dispatcher
}

func BindEvent(name string, handler EventHandler[util.M]) {
	_Dispatcher.Bind(name, handler)
}

func UnbindEvent(name string, handler EventHandler[util.M]) {
	_Dispatcher.Unbind(name, handler)
}

func DispatchEvent(name string, data any) {
	_Dispatcher.Dispatch(name, data)
}

func NewDispatcher[T any](data T) *Dispatcher[T] {
	return &Dispatcher[T]{
		data:          data,
		nameToHandler: make(map[string]*ds.Link[EventHandler[T]]),
	}
}

type EventHandler[T any] func(T, any)

type Dispatcher[T any] struct {
	data          T
	nameToHandler map[string]*ds.Link[EventHandler[T]]
}

func (d *Dispatcher[T]) Bind(name string, handler EventHandler[T]) {
	link, ok := d.nameToHandler[name]
	if !ok {
		link = ds.NewLink[EventHandler[T]]()
		d.nameToHandler[name] = link
	}
	link.Push(handler)
}

func (d *Dispatcher[T]) Unbind(name string, handler EventHandler[T]) {
	link, ok := d.nameToHandler[name]
	if !ok {
		return
	}
	pointer := reflect.ValueOf(handler).Pointer()
	_ = link.Del(func(fn EventHandler[T]) bool {
		return reflect.ValueOf(fn).Pointer() == pointer
	})
}

func (d *Dispatcher[T]) Dispatch(name string, data any) {
	link, ok := d.nameToHandler[name]
	if !ok {
		return
	}
	link.Iter(func(handler EventHandler[T]) {
		handler(d.data, data)
	})
}
