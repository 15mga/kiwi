package worker

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync"
)

type buffer struct {
	items []any
	cap   int
	count int
}

func (b *buffer) Push(item any) {
	if b.count == b.cap {
		b.cap = b.count << 1
		items := make([]any, b.cap)
		copy(items, b.items)
		b.items = items
	}
	b.items[b.count] = item
	b.count++
}

func NewWorker(c int, fn func(any)) *Worker {
	b := &Worker{
		sign: make(chan struct{}, 1),
		fn:   fn,
		swap: &buffer{
			items: make([]any, c),
			cap:   c,
		},
		buffer: &buffer{
			items: make([]any, c),
			cap:   c,
		},
	}
	return b
}

type Worker struct {
	fn     func(any)
	buffer *buffer
	swap   *buffer
	mtx    sync.Mutex
	sign   chan struct{}
	idx    int
}

func (p *Worker) Start() {
	go p.start()
}

func (p *Worker) Dispose() {
	close(p.sign)
}

func (p *Worker) start() {
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

func (p *Worker) Push(item any) {
	p.mtx.Lock()
	p.buffer.Push(item)
	p.mtx.Unlock()

	select {
	case p.sign <- struct{}{}:
	default:
	}
}

func (p *Worker) do() {
	if p.swap.count == 0 {
		return
	}
	items := p.swap.items
	for i, item := range items[p.idx:p.swap.count] {
		p.idx++
		items[i] = nil
		p.fn(item)
	}
	p.swap.count = 0
	p.idx = 0
}
