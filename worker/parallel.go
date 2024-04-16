package worker

import (
	"github.com/15mga/kiwi/ds"
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	_Parallel    *parallel
	_ParallelNum int
	_WorkerNum   int
	_WorkerNum32 uint32
)

func init() {
	_ParallelNum = runtime.NumCPU()
	if _ParallelNum < 8 {
		_ParallelNum = 8
	}
	_WorkerNum = _ParallelNum - 1
	_WorkerNum32 = uint32(_WorkerNum)
}

func WorkerNum() int {
	return _WorkerNum
}

type parallel struct {
	workers []*parallelWorker
}

func InitParallel() {
	if _Parallel != nil {
		return
	}
	_Parallel = &parallel{
		workers: make([]*parallelWorker, _WorkerNum),
	}
	for i := 0; i < _WorkerNum; i++ {
		w := newParallelWorker()
		_Parallel.workers[i] = w
		go w.start()
	}
}

var (
	_WorkerIdx uint32
)

func PushPJob(job IJob) {
	idx := atomic.AddUint32(&_WorkerIdx, 1)
	_Parallel.workers[idx%_WorkerNum32].PushJob(job)
}

func GetAvgCount(l, min int) int {
	if l < min*_WorkerNum {
		return min
	}
	num := _ParallelNum
	count := l / num
	if l%num != 0 {
		count++
	}
	return count
}

func newParallelWorker() *parallelWorker {
	return &parallelWorker{
		jobCh: make(chan IJob, 32),
	}
}

type parallelWorker struct {
	jobCh chan IJob
}

func (w *parallelWorker) PushJob(job IJob) {
	w.jobCh <- job
}

func (w *parallelWorker) start() {
	for j := range w.jobCh {
		j.Do()
	}
}

func P[T any](jobPool *sync.Pool, min int, items []T, fn func(T)) {
	l := len(items)
	if l <= min {
		for _, c := range items {
			fn(c)
		}
		return
	}
	var wg sync.WaitGroup
	avg := GetAvgCount(l, min)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		j := jobPool.Get().(*PJob[T])
		j.start = start
		j.end = end
		j.wg = &wg
		j.items = items
		j.fn = fn
		j.pool = jobPool
		PushPJob(j)
	}
	for _, item := range items[:avg] {
		fn(item)
	}
	wg.Wait()
}

func PFilter[T comparable](jobPool *sync.Pool, min int, items []T, filter func(T) bool, fn func([]T)) {
	l := len(items)
	if l <= min {
		arr := ds.NewArray[T](l)
		for _, item := range items {
			if filter(item) {
				arr.Add(item)
			}
		}
		if arr.Count() > 0 {
			fn(arr.Values())
		}
		return
	}
	var wg sync.WaitGroup
	all := make([]*ds.Array[T], 0, _WorkerNum)
	avg := GetAvgCount(l, min)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		j := jobPool.Get().(*PFilterJob[T])
		j.start = start
		j.end = end
		j.wg = &wg
		j.pool = jobPool
		j.items = items
		j.filter = filter
		j.Array.Reset()
		all = append(all, j.Array)
		PushPJob(j)
	}
	arr := ds.NewArray[T](avg)
	for _, item := range items[:avg] {
		if filter(item) {
			arr.Add(item)
		}
	}
	if arr.Count() > 0 {
		fn(arr.Values())
		arr.Reset()
	}
	wg.Wait()

	for _, a := range all {
		if a.Count() > 0 {
			fn(a.Values())
		}
	}
}

func PAfter[T comparable](jobPool *sync.Pool, min int, items []T, fnItem func(T, *ds.FnLink)) {
	l := len(items)
	if l <= min {
		link := ds.NewFnLink()
		for _, c := range items {
			fnItem(c, link)
		}
		link.Invoke()
		return
	}
	var wg sync.WaitGroup
	avg := GetAvgCount(l, min)
	var end int
	jobs := make([]*PAfterJob[T], 0, _WorkerNum)
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		j := jobPool.Get().(*PAfterJob[T])
		j.start = start
		j.end = end
		j.wg = &wg
		j.pool = jobPool
		j.items = items
		j.fnItem = fnItem
		j.Link.Reset()
		jobs = append(jobs, j)
		PushPJob(j)
	}
	link := ds.NewFnLink()
	for _, item := range items[:avg] {
		fnItem(item, link)
	}
	link.Invoke()
	wg.Wait()
	for _, j := range jobs {
		j.Link.Invoke()
	}
}

type IJob interface {
	Do()
}

type baseJob struct {
	start, end int
	wg         *sync.WaitGroup
	pool       *sync.Pool
}

type PJob[T any] struct {
	baseJob
	items []T
	fn    func(T)
}

func (j *PJob[T]) Do() {
	for _, item := range j.items[j.start:j.end] {
		j.fn(item)
	}
	j.wg.Done()
	j.pool.Put(j)
}

type PFilterJob[T comparable] struct {
	baseJob
	items  []T
	Array  *ds.Array[T]
	filter func(T) bool
}

func (j *PFilterJob[T]) Do() {
	for _, item := range j.items[j.start:j.end] {
		if j.filter(item) {
			j.Array.Add(item)
		}
	}
	j.wg.Done()
	j.pool.Put(j)
}

type PAfterJob[T comparable] struct {
	baseJob
	fnItem func(T, *ds.FnLink)
	items  []T
	Link   *ds.FnLink
}

func (j *PAfterJob[T]) Do() {
	for _, item := range j.items[j.start:j.end] {
		j.fnItem(item, j.Link)
	}
	j.wg.Done()
	j.pool.Put(j)
}
