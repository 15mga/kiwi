package worker

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync"
)

type fnWorkData struct {
	fn   util.FnAny
	data any
}

type fnWorkBuffer struct {
	items []*fnWorkData
	cap   int
	count int
}

func (b *fnWorkBuffer) Push(fn util.FnAny, data any) {
	if b.count == b.cap {
		b.cap = b.count << 1
		items := make([]*fnWorkData, b.cap)
		copy(items, b.items)
		b.items = items
	}
	if b.items[b.count] == nil {
		b.items[b.count] = &fnWorkData{
			fn:   fn,
			data: data,
		}
	} else {
		b.items[b.count].fn = fn
		b.items[b.count].data = data
	}
	b.count++
}

func NewFnWorker(c int) *FnWorker {
	b := &FnWorker{
		ch: make(chan struct{}, 1),
		swap: &fnWorkBuffer{
			items: make([]*fnWorkData, c),
			cap:   c,
		},
		buffer: &fnWorkBuffer{
			items: make([]*fnWorkData, c),
			cap:   c,
		},
	}
	return b
}

type FnWorker struct {
	buffer *fnWorkBuffer
	swap   *fnWorkBuffer
	mtx    sync.Mutex
	ch     chan struct{}
	idx    int
}

func (p *FnWorker) Start() {
	go p.start()
}

func (p *FnWorker) Dispose() {
	close(p.ch)
}

func (p *FnWorker) start() {
	defer func() {
		if err := recover(); err != nil {
			kiwi.Error2(util.EcServiceErr, util.M{
				"error": fmt.Sprint(err),
			})
			p.Start()
		}
	}()

	p.do()

	for range p.ch {
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

func (p *FnWorker) Push(fn util.FnAny, data any) {
	p.mtx.Lock()
	p.buffer.Push(fn, data)
	p.mtx.Unlock()

	select {
	case p.ch <- struct{}{}:
	default:
	}
}

func (p *FnWorker) do() {
	if p.swap.count == 0 {
		return
	}
	items := p.swap.items
	for _, item := range items[p.idx:p.swap.count] {
		p.idx++
		item.fn(item.data)
		item.fn = nil
		item.data = nil
	}
	p.swap.count = 0
	p.idx = 0
}
