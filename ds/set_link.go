package ds

import (
	"github.com/15mga/kiwi/util"
)

type (
	setLinkOption[KT comparable, VT any] struct {
		valToKey    func(VT) KT
		spawnElem   func() *SetLinkElem[VT]
		recycleElem func(*SetLinkElem[VT])
	}
	DLinkOption[KT comparable, VT any] func(o *setLinkOption[KT, VT])
)

func SetLinkValToKey[KT comparable, VT any](valToKey func(VT) KT) DLinkOption[KT, VT] {
	return func(o *setLinkOption[KT, VT]) {
		o.valToKey = valToKey
	}
}

func SetLinkSpawnElem[KT comparable, VT any](spawn func() *SetLinkElem[VT]) DLinkOption[KT, VT] {
	return func(o *setLinkOption[KT, VT]) {
		o.spawnElem = spawn
	}
}

func SetLinkRecycleElem[KT comparable, VT any](recycle func(*SetLinkElem[VT])) DLinkOption[KT, VT] {
	return func(o *setLinkOption[KT, VT]) {
		o.recycleElem = recycle
	}
}

func NewSetLink[KT comparable, VT any](cap int, opts ...DLinkOption[KT, VT]) *SetLink[KT, VT] {
	opt := &setLinkOption[KT, VT]{
		spawnElem: func() *SetLinkElem[VT] {
			return &SetLinkElem[VT]{}
		},
		recycleElem: func(elem *SetLinkElem[VT]) {},
	}
	for _, o := range opts {
		o(opt)
	}
	return &SetLink[KT, VT]{
		option:    opt,
		keyToNode: make(map[KT]*SetLinkElem[VT], cap),
	}
}

// SetLink 元素唯一双向链表
type SetLink[KT comparable, VT any] struct {
	option    *setLinkOption[KT, VT]
	head      *SetLinkElem[VT]
	tail      *SetLinkElem[VT]
	keyToNode map[KT]*SetLinkElem[VT]
	count     int
}

func (l *SetLink[KT, VT]) Get(key KT) (VT, bool) {
	v, ok := l.keyToNode[key]
	if ok {
		return v.Value, true
	}
	return util.Default[VT](), false
}

// Push 推入元素，如果已存在返回false
func (l *SetLink[KT, VT]) Push(val VT) bool {
	key := l.option.valToKey(val)
	_, ok := l.keyToNode[key]
	if ok {
		return false
	}
	l.push(key, val)
	return true
}

func (l *SetLink[KT, VT]) push(key KT, val VT) {
	node := l.option.spawnElem()
	node.Value = val
	l.keyToNode[key] = node
	if l.count == 0 {
		l.head = node
		l.tail = node
	} else {
		node.prevNode = l.tail
		l.tail.nextNode = node
		l.tail = node
	}
	l.count++
}

func (l *SetLink[KT, VT]) NewOrUpdate(key KT, update func(VT), new func() VT) (exist bool) {
	v, ok := l.keyToNode[key]
	if ok {
		update(v.Value)
		return true
	}
	l.push(key, new())
	return false
}

// Del 测试弹出一个符合条件的即停止
func (l *SetLink[KT, VT]) Del(test func(VT) bool) (VT, bool) {
	for node := l.head; node != nil; node = node.nextNode {
		if test(node.Value) {
			v := node.Value
			l.removeNode(node)
			return v, true
		}
	}
	return util.Default[VT](), false
}

func (l *SetLink[KT, VT]) Pop() (VT, bool) {
	if l.count == 0 {
		return util.Default[VT](), false
	}
	firstNode := l.head
	v := firstNode.Value
	l.head = firstNode.nextNode
	if l.option.recycleElem != nil {
		firstNode.Dispose()
		l.option.recycleElem(firstNode)
	}
	delete(l.keyToNode, l.option.valToKey(v))
	return v, true
}

func (l *SetLink[KT, VT]) removeNode(node *SetLinkElem[VT]) {
	if l.head == node {
		if l.tail == node {
			l.head = nil
			l.tail = nil
		} else {
			l.head = node.nextNode
			l.head.prevNode = nil
		}
	} else if l.tail == node {
		prevNode := node.prevNode
		prevNode.nextNode = nil
		l.tail = prevNode
	} else {
		prevNode := node.prevNode
		nextNode := node.nextNode
		prevNode.nextNode = nextNode
		nextNode.prevNode = prevNode
	}
	delete(l.keyToNode, l.option.valToKey(node.Value))
	l.count--
	if l.option.recycleElem != nil {
		node.Dispose()
		l.option.recycleElem(node)
	}
}

func (l *SetLink[KT, VT]) Remove(val VT) {
	l.RemoveByKey(l.option.valToKey(val))
}

func (l *SetLink[KT, VT]) RemoveByKey(key KT) {
	node, ok := l.keyToNode[key]
	if !ok {
		return
	}
	l.removeNode(node)
}

func (l *SetLink[KT, VT]) Iter(fn func(VT)) {
	if l.count == 0 {
		return
	}
	for n := l.head; n != nil; n = n.nextNode {
		fn(n.Value)
	}
}

func (l *SetLink[KT, VT]) Any(fn func(VT) bool) bool {
	if l.count == 0 {
		return false
	}
	for n := l.head; n != nil; n = n.nextNode {
		if fn(n.Value) {
			return true
		}
	}
	return false
}

func (l *SetLink[KT, VT]) Values() []VT {
	if l.count == 0 {
		return nil
	}
	values := make([]VT, l.count)
	i := 0
	for n := l.head; n != nil; n = n.nextNode {
		values[i] = n.Value
		i++
	}
	return values
}

func (l *SetLink[KT, VT]) Count() int {
	return l.count
}

type SetLinkElem[T any] struct {
	Value    T
	prevNode *SetLinkElem[T]
	nextNode *SetLinkElem[T]
}

func (n *SetLinkElem[T]) Dispose() {
	n.prevNode = nil
	n.nextNode = nil
}
