package worker

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"sync"
	"testing"
)

type movement struct {
	pos, dir util.Vec2
	speed    float32
	wg       *sync.WaitGroup
}

type data struct {
	name string
	data any
}

func testFn(a any) {
	switch m := a.(type) {
	case *movement:
		p := util.Vec2Add(m.pos, util.Vec2Mul(m.dir, m.speed))
		m.pos.X = p.X
		m.pos.Y = p.Y
		m.wg.Done()
	}
}

func testFn2(d *data) {
	switch d.name {
	case "movement":
		m := d.data.(*movement)
		p := util.Vec2Add(m.pos, util.Vec2Mul(m.dir, m.speed))
		m.pos.X = p.X
		m.pos.Y = p.Y
		m.wg.Done()
	}
}

func BenchmarkWorker(b *testing.B) {
	channel := newDefWork(testFn)
	channel.Start()
	worker := NewWorker(8096, testFn)
	worker.Start()
	worker2 := NewWorker2[*data](8096, testFn2)
	worker2.Start()

	wg := sync.WaitGroup{}
	m := &movement{
		pos:   util.Vec2{},
		dir:   util.Vec2{1, 0},
		speed: 1,
		wg:    &wg,
	}
	d := &data{
		name: "movement",
		data: m,
	}
	countSlc := []int{100, 500, 1000, 2000, 5000}
	for _, count := range countSlc {
		b.Run(fmt.Sprintf("chan_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				wg.Add(count)
				for j := 0; j < count; j++ {
					channel.Push(m)
				}
				wg.Wait()
			}
		})
		b.Run(fmt.Sprintf("worker_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				wg.Add(count)
				for j := 0; j < count; j++ {
					worker.Push(m)
				}
				wg.Wait()
			}
		})
		b.Run(fmt.Sprintf("worker2_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				wg.Add(count)
				for j := 0; j < count; j++ {
					worker2.Push(d)
				}
				wg.Wait()
			}
		})
	}
}

func newDefWork(fn func(any)) *defWorker {
	return &defWorker{
		ch: make(chan any, 1),
		fn: fn,
	}
}

type defWorker struct {
	ch chan any
	fn func(any)
}

func (w *defWorker) Start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				//fmt.Printf("\033[31m!!!recover!!!\u001B[0m\n%s%s\n", err, util.GetStack(5))
				kiwi.Error2(util.EcServiceErr, util.M{
					"error": fmt.Sprint(err),
				})
				w.Start()
			}
		}()

		for item := range w.ch {
			w.fn(item)
		}
	}()
}

func (w *defWorker) Push(item any) {
	w.ch <- item
}
