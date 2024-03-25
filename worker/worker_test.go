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

func testFn(d *movement) {
	p := util.Vec2Add(d.pos, util.Vec2Mul(d.dir, d.speed))
	d.pos.X = p.X
	d.pos.Y = p.Y
	d.wg.Done()
}

func BenchmarkWorker(b *testing.B) {
	dw := newDefWork[*movement](testFn)
	dw.Start()
	pool := &sync.Pool{
		New: func() any {
			return &job[*movement]{}
		},
	}
	w := NewWorker[*movement](testFn, pool)
	w.Start()

	wg := sync.WaitGroup{}
	count := 1000
	m := &movement{
		pos:   util.Vec2{},
		dir:   util.Vec2{1, 0},
		speed: 1,
		wg:    &wg,
	}
	b.Run("def chan", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			wg.Add(count)
			for j := 0; j < count; j++ {
				dw.Push(m)
			}
			wg.Wait()
		}
	})
	m = &movement{
		pos:   util.Vec2{},
		dir:   util.Vec2{1, 0},
		speed: 1,
		wg:    &wg,
	}
	b.Run("worker", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			wg.Add(count)
			for j := 0; j < 1000; j++ {
				w.Push(m)
			}
			wg.Wait()
		}
	})
}

func newDefWork[T any](fn func(T)) *defWorker[T] {
	return &defWorker[T]{
		ch: make(chan T, 1),
		fn: fn,
	}
}

type defWorker[T any] struct {
	ch chan T
	fn func(T)
}

func (w *defWorker[T]) Start() {
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

func (w *defWorker[T]) Push(item T) {
	w.ch <- item
}
