package worker

import (
	"fmt"
	"github.com/15mga/kiwi"
	"sync"

	"github.com/15mga/kiwi/util"
)

func NewWorker[T any](fn func(T)) *Worker[T] {
	b := &Worker[T]{
		ch: make(chan struct{}, 1),
		fn: fn,
		pool: sync.Pool{
			New: func() any {
				return &job[T]{}
			},
		},
	}
	return b
}

type Worker[T any] struct {
	mtx  sync.Mutex
	ch   chan struct{}
	head *job[T]
	tail *job[T]
	curr *job[T]
	fn   func(T)
	pool sync.Pool
}

func (w *Worker[T]) Start() {
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

		w.do()

		for range w.ch {
			for {
				w.mtx.Lock()
				w.curr = w.head
				w.head = nil
				w.tail = nil
				w.mtx.Unlock()

				if w.curr == nil {
					break
				}
				w.do()
			}
		}
	}()
}

func (w *Worker[T]) Dispose() {
	close(w.ch)
}

func (w *Worker[T]) Push(item T) {
	e := w.pool.Get().(*job[T])
	e.value = item
	w.mtx.Lock()
	if w.head != nil {
		w.tail.next = e
	} else {
		w.head = e
	}
	w.tail = e
	w.mtx.Unlock()

	select {
	case w.ch <- struct{}{}:
	default:
	}
}

func (w *Worker[T]) do() {
	var (
		j   *job[T]
		val T
	)
	for {
		if w.curr == nil {
			break
		}
		j = w.curr
		val = j.value
		w.curr = j.next
		j.next = nil
		w.pool.Put(j)
		w.fn(val)
	}
}

type job[T any] struct {
	next  *job[T]
	value T
}

type FnJob func(*Job)

func NewJobWorker(fn FnJob) *JobWorker {
	w := &JobWorker{
		pcr: fn,
	}
	w.Worker = NewWorker[*Job](w.process)
	return w
}

type JobWorker struct {
	*Worker[*Job]
	pcr FnJob
}

func (w *JobWorker) process(j *Job) {
	w.pcr(j)
	j.Data = nil
	_JobPool.Put(j)
}

func (w *JobWorker) Push(name JobName, data ...any) {
	j := _JobPool.Get().(*Job)
	j.Name = name
	j.Data = data
	w.Worker.Push(j)
}

type JobName = string

type Job struct {
	Name JobName
	Data []any
}

var (
	_JobPool = sync.Pool{
		New: func() any {
			return &Job{}
		},
	}
)

func SpawnJob() *Job {
	return _JobPool.Get().(*Job)
}

func RecycleJob(job *Job) {
	_JobPool.Put(job)
}

type FnJobData struct {
	Fn     util.FnAnySlc
	Params []any
}

type FnWorker struct {
	*Worker[FnJobData]
}

func NewFnWorker() *FnWorker {
	return &FnWorker{
		Worker: NewWorker[FnJobData](func(data FnJobData) {
			data.Fn(data.Params)
		}),
	}
}

func (w *FnWorker) Push(fn util.FnAnySlc, params ...any) {
	w.Worker.Push(FnJobData{
		Fn:     fn,
		Params: params,
	})
}
