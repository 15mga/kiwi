package util

import (
	"sync"
	"sync/atomic"
)

var (
	_CounterPool = sync.Pool{
		New: func() any {
			return &Counter{}
		},
	}
)

func SpawnCounter(total uint32) *Counter {
	c := _CounterPool.Get().(*Counter)
	c.total = total
	return c
}

func RecycleCounter(c *Counter) {
	_CounterPool.Put(c)
}

func NewCounter(total uint32) *Counter {
	return &Counter{
		total: total,
	}
}

type Counter struct {
	total uint32
	count uint32
}

func (c *Counter) Total() uint32 {
	return c.total
}

func (c *Counter) Count() uint32 {
	return c.count
}

func (c *Counter) Reset(total uint32) {
	c.count = 0
	c.total = total
}

func (c *Counter) Add(v uint32) (uint32, bool) {
	n := atomic.AddUint32(&c.count, v)
	return n, n == c.total
}
