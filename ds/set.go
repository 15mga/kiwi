package ds

import (
	"github.com/15mga/kiwi/util"
)

func NewKSet[KT comparable, VT any](defCap int, getKey func(VT) KT) *KSet[KT, VT] {
	if defCap == 0 {
		defCap = 1
	}
	return &KSet[KT, VT]{
		items:    make([]VT, defCap),
		keyToIdx: make(map[KT]int, defCap),
		cap:      defCap,
		defCap:   defCap,
		getKey:   getKey,
		defVal:   util.Default[VT](),
	}
}

type KSet[KT comparable, VT any] struct {
	items    []VT
	keyToIdx map[KT]int
	count    int
	cap      int
	defCap   int
	getKey   func(VT) KT
	defVal   VT
}

func (s *KSet[KT, VT]) Count() int {
	return s.count
}

func (s *KSet[KT, VT]) Cap() int {
	return s.cap
}

func (s *KSet[KT, VT]) Add(item VT) *util.Err {
	key := s.getKey(item)
	_, ok := s.keyToIdx[key]
	if ok {
		return util.NewErr(util.EcExist, util.M{
			"key": key,
		})
	}
	s.add(key, item)
	return nil
}

func (s *KSet[KT, VT]) AddNX(item VT) bool {
	key := s.getKey(item)
	_, ok := s.keyToIdx[key]
	if ok {
		return false
	}
	s.add(key, item)
	return true
}

func (s *KSet[KT, VT]) AddNX2(key KT, new func() VT) bool {
	_, ok := s.keyToIdx[key]
	if ok {
		return false
	}
	v := new()
	s.add(key, v)
	return true
}

func (s *KSet[KT, VT]) add(key KT, item VT) {
	s.testGrow()
	s.items[s.count] = item
	s.keyToIdx[key] = s.count
	s.count++
}

func (s *KSet[KT, VT]) testGrow() {
	if s.count+1 < s.cap {
		return
	}
	if s.cap < 1024 {
		s.cap = s.cap << 1
	} else {
		s.cap *= 2
	}
	ns := make([]VT, s.cap)
	copy(ns, s.items)
	s.items = ns
}

func (s *KSet[KT, VT]) testShrink() {
	// 缩容
	if s.cap == s.defCap {
		return
	}
	var h int
	if s.cap < 1024 {
		h = s.cap >> 1
	} else {
		h = s.cap / 2
	}
	if s.count > h {
		return
	}
	ns := make([]VT, h)
	copy(ns, s.items[:s.count])
	s.items = ns
	s.cap = h
	nm := make(map[KT]int, s.count)
	for k, v := range s.keyToIdx {
		nm[k] = v
	}
	s.keyToIdx = nm
}

func (s *KSet[KT, VT]) Set(item VT) (old VT) {
	key := s.getKey(item)
	idx, ok := s.keyToIdx[key]
	if ok {
		old = s.items[idx]
		s.items[idx] = item
		return
	}
	s.add(key, item)
	return
}

func (s *KSet[KT, VT]) Del(k KT) (val VT, exist bool) {
	idx, ok := s.keyToIdx[k]
	if !ok {
		return
	}
	val = s.items[idx]
	exist = true
	delete(s.keyToIdx, k)
	c := s.count - 1
	if idx == c || c == 0 {
		s.items[idx] = s.defVal
	} else {
		tail := s.items[c]
		s.items[idx] = tail
		s.items[c] = s.defVal
		s.keyToIdx[s.getKey(tail)] = idx
	}
	s.count = c
	s.testShrink()
	return
}

func (s *KSet[KT, VT]) Reset() {
	for i := 0; i < s.count; i++ {
		s.items[i] = s.defVal
	}
	s.count = 0
	s.keyToIdx = make(map[KT]int, s.defCap)
}

func (s *KSet[KT, VT]) ReplaceOrNew(oldKey KT, newItem VT) bool {
	idx, ok := s.keyToIdx[oldKey]
	if !ok {
		_ = s.Add(newItem)
		return false
	}
	delete(s.keyToIdx, oldKey)
	s.keyToIdx[s.getKey(newItem)] = idx
	s.items[idx] = newItem
	return true
}

func (s *KSet[KT, VT]) Get(key KT) (VT, bool) {
	idx, ok := s.keyToIdx[key]
	if !ok {
		return s.defVal, false
	}
	item := s.items[idx]
	return item, true
}

func (s *KSet[KT, VT]) GetOrNew(key KT, new func() VT) (VT, bool) {
	idx, ok := s.keyToIdx[key]
	if ok {
		return s.items[idx], true
	}
	n := new()
	s.add(key, n)
	return n, false
}

func (s *KSet[KT, VT]) GetWithIdx(idx int) (VT, bool) {
	if idx >= s.count || idx < 0 {
		return s.defVal, false
	}
	item := s.items[idx]
	return item, true
}

func (s *KSet[KT, VT]) Has(key KT) bool {
	_, ok := s.keyToIdx[key]
	return ok
}

func (s *KSet[KT, VT]) Iter(fn func(VT)) {
	for i := 0; i < s.count; i++ {
		fn(s.items[i])
	}
}

func (s *KSet[KT, VT]) Any(fn func(VT) bool) bool {
	for i := 0; i < s.count; i++ {
		item := s.items[i]
		if fn(item) {
			return true
		}
	}
	return false
}

func (s *KSet[KT, VT]) Values() []VT {
	return s.items[:s.count]
}

func (s *KSet[KT, VT]) CopyValues(values *[]VT) {
	for _, v := range s.items[:s.count] {
		*values = append(*values, v)
	}
}

func (s *KSet[KT, VT]) CopyKeys(keys *[]KT) {
	for k := range s.keyToIdx {
		*keys = append(*keys, k)
	}
}

func (s *KSet[KT, VT]) TestDel(test func(KT, VT) (del bool, brk bool)) {
	for i := s.count - 1; i > -1; i-- {
		item := s.items[i]
		key := s.getKey(item)
		if ok, brk := test(key, item); ok {
			s.Del(key)
			if !brk {
				break
			}
		}
	}
}

func NewSet[KT comparable, VT any](defCap int) *KSet[KT, *SetItem[KT, VT]] {
	return NewKSet[KT, *SetItem[KT, VT]](defCap, func(k *SetItem[KT, VT]) KT {
		return k.key
	})
}

func NewSetItem[KT comparable, VT any](defCap int, key KT, getKey func(VT) KT) *SetItem[KT, VT] {
	return &SetItem[KT, VT]{
		key:  key,
		KSet: NewKSet[KT, VT](defCap, getKey),
	}
}

type SetItem[KT comparable, VT any] struct {
	key KT
	*KSet[KT, VT]
}

func (s *SetItem[KT, VT]) ResetKey(key KT) {
	s.key = key
}

func NewSet2Item[KT1, KT2 comparable, VT any](key KT1, defCap int, getKey func(VT) KT2) *Set2Item[KT1, KT2, VT] {
	return &Set2Item[KT1, KT2, VT]{
		key:  key,
		KSet: NewKSet[KT2, VT](defCap, getKey),
	}
}

type Set2Item[KT1, KT2 comparable, VT any] struct {
	*KSet[KT2, VT]
	key KT1
}

func (s *Set2Item[KT1, KT2, VT]) Key() KT1 {
	return s.key
}

func NewKSet2[KT1, KT2 comparable, VT any](defCap int, getKey func(VT) KT2) *KSet2[KT1, KT2, VT] {
	return &KSet2[KT1, KT2, VT]{
		defCap: defCap,
		getKey: getKey,
		KSet: NewKSet[KT1, *Set2Item[KT1, KT2, VT]](defCap, func(s *Set2Item[KT1, KT2, VT]) KT1 {
			return s.key
		}),
	}
}

type KSet2[KT1, KT2 comparable, VT any] struct {
	*KSet[KT1, *Set2Item[KT1, KT2, VT]]
	getKey func(VT) KT2
	defCap int
}

func (s *KSet2[KT1, KT2, VT]) ReplaceKey(old, new KT1) bool {
	idx, ok := s.keyToIdx[old]
	if !ok {
		return false
	}
	s.keyToIdx[new] = idx
	s.items[idx].key = new
	return true
}
