package ds

import (
	"github.com/15mga/kiwi/util"
)

func newArrayBase[T comparable](defCap int) arrayBase[T] {
	return arrayBase[T]{
		items:  make([]T, defCap),
		defCap: defCap,
		defVal: util.Default[T](),
	}
}

type arrayBase[T comparable] struct {
	items  []T
	count  int
	defCap int
	defVal T
}

func (a *arrayBase[T]) Count() int {
	return a.count
}

func (a *arrayBase[T]) Add(item T) {
	a.testGrow(1)
	a.items[a.count] = item
	a.count++
}

func (a *arrayBase[T]) AddRange(items []T) {
	l := len(items)
	if l == 0 {
		return
	}
	a.testGrow(l)
	for i, item := range items {
		a.items[a.count+i] = item
	}
	a.count += l
}

//func (a *arrayBase[T]) Del(v T) (exist bool) {
//	panic("not implement")
//}

func (a *arrayBase[T]) testGrow(n int) {
	c, ok := util.NextCap(a.count+n, len(a.items), 1024)
	if !ok {
		return
	}
	ns := make([]T, c)
	copy(ns, a.items)
	a.items = ns
}

func (a *arrayBase[T]) Clean() {
	a.count = 0
}

func (a *arrayBase[T]) Reset() {
	if a.count == 0 {
		return
	}
	l := len(a.items)
	h := l >> 1
	if a.count < h {
		a.items = make([]T, h)
	} else {
		for i := 0; i < a.count; i++ {
			a.items[i] = a.defVal
		}
	}
	a.count = 0
}

func (a *arrayBase[T]) Values() []T {
	return a.items[:a.count]
}

func (a *arrayBase[T]) HasItem(item T) bool {
	for i := 0; i < a.count; i++ {
		if a.items[i] == item {
			return true
		}
	}
	return false
}

func (a *arrayBase[T]) GetItem(idx int) T {
	return a.items[idx]
}

func NewArray[T comparable](defCap int) *Array[T] {
	return &Array[T]{
		arrayBase: newArrayBase[T](defCap),
	}
}

// Array 无序数组
type Array[T comparable] struct {
	arrayBase[T]
}

func (a *Array[T]) Del(v T) (exist bool) {
	var idx int
	for i, item := range a.items {
		if item == v {
			exist = true
			idx = i
			break
		}
	}
	if !exist {
		return
	}
	c := a.count - 1
	if idx == c || c == 0 {
		a.items[idx] = a.defVal
		a.count = c
	} else {
		tail := a.items[c]
		a.items[idx] = tail
		a.items[c] = a.defVal
		a.count = c
		a.testShrink()
	}
	return
}

func (a *Array[T]) testShrink() {
	c := len(a.items)
	if c == a.defCap {
		return
	}
	var l int
	if c < 1024 {
		l = c >> 1
	} else {
		l = c / 2
	}
	if a.count > l {
		return
	}
	ns := make([]T, l)
	copy(ns, a.items)
	a.items = ns
}

func NewList[T comparable](defCap int) *List[T] {
	return &List[T]{
		arrayBase: newArrayBase[T](defCap),
	}
}

// List 有序数组
type List[T comparable] struct {
	arrayBase[T]
}

func (l *List[T]) Del(v T) (exist bool) {
	var idx int
	for i, item := range l.items {
		if item == v {
			exist = true
			idx = i
			break
		}
	}
	if !exist {
		return
	}
	c := l.count - 1
	if idx == c || c == 0 {
		l.items[idx] = l.defVal
	} else {
		if n, ok := l.isNeedShrink(); ok {
			ns := make([]T, n)
			copy(ns, l.items[:idx])
			copy(ns[idx:], l.items[idx+1:])
		} else {
			//copy(l.items[:idx], l.items[idx+1:])
			l.items = append(l.items[:idx], l.items[idx+1:]...)
		}
	}
	l.count = c
	return
}

func (l *List[T]) isNeedShrink() (int, bool) {
	c := len(l.items)
	if c == l.defCap {
		return 0, false
	}

	var n int
	if c < 1024 {
		n = c >> 1
	} else {
		n = c / 2
	}
	if l.count > n {
		return 0, false
	}
	return n, true
}
