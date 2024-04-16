package worker

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync"
)

type buffer2[T comparable] struct {
	items []T
	cap   int
	count int
}

func (b *buffer2[T]) Push(item T) {
	if b.count == b.cap {
		b.cap = b.count << 1
		items := make([]T, b.cap)
		copy(items, b.items)
		b.items = items
	}
	b.items[b.count] = item
	b.count++
}

func NewWorker2[T comparable](c int, fn func(T)) *Worker2[T] {
	b := &Worker2[T]{
		sign: make(chan struct{}, 1),
		fn:   fn,
		swap: &buffer2[T]{
			items: make([]T, c),
			cap:   c,
		},
		buffer: &buffer2[T]{
			items: make([]T, c),
			cap:   c,
		},
	}
	return b
}

type Worker2[T comparable] struct {
	fn     func(T)
	buffer *buffer2[T]
	swap   *buffer2[T]
	mtx    sync.Mutex
	sign   chan struct{}
	idx    int
}

func (p *Worker2[T]) Start() {
	go p.start()
}

func (p *Worker2[T]) Dispose() {
	close(p.sign)
}

func (p *Worker2[T]) start() {
	defer func() {
		if err := recover(); err != nil {
			kiwi.Error2(util.EcServiceErr, util.M{
				"error": fmt.Sprint(err),
			})
			p.Start()
		}
	}()

	p.do()

	for range p.sign {
		for {
			p.mtx.Lock()
			if p.buffer.count == 0 {
				p.mtx.Unlock()
				break
			}
			p.swap, p.buffer = p.buffer, p.swap
			p.mtx.Unlock()

			p.do()
		}
	}
}

func (p *Worker2[T]) Push(item T) {
	p.mtx.Lock()
	p.buffer.Push(item)
	p.mtx.Unlock()

	select {
	case p.sign <- struct{}{}:
	default:
	}
}

func (p *Worker2[T]) do() {
	if p.swap.count == 0 {
		return
	}
	items := p.swap.items
	for i, item := range items[p.idx:p.swap.count] {
		p.idx++
		items[i] = util.Default[T]()
		p.fn(item)
	}
	p.swap.count = 0
	p.idx = 0
}
