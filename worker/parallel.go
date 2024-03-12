package worker

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

var (
	_Parallel         *parallel
	_ParallelNum      int
	_ParallelNum32    uint32
	_JobParallelCount int
)

func init() {
	n := runtime.NumCPU()
	if n < 8 {
		n = 8
	}
	_ParallelNum = n
	_ParallelNum32 = uint32(n)
	_JobParallelCount = _JobUnit * n
}

const (
	_JobUnit = 64
)

type parallel struct {
	workers []*parallelWorker
}

func InitParallel() {
	if _Parallel != nil {
		return
	}
	_Parallel = &parallel{
		workers: make([]*parallelWorker, _ParallelNum),
	}
	for i := 0; i < _ParallelNum; i++ {
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
	_Parallel.workers[idx%_ParallelNum32].PushJob(job)
}

func getAvgCount(l int) int {
	if l < _JobParallelCount {
		return _JobUnit
	}
	count := l / _ParallelNum
	if l%_ParallelNum != 0 {
		count++
	}
	return count
}

func PFn(fns []util.FnAnySlc, params ...any) {
	l := len(fns)
	if l <= _JobUnit {
		for _, fn := range fns {
			fn(params)
		}
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		PushPJob(&fnJob{
			fns:    fns,
			start:  start,
			end:    end,
			wg:     &wg,
			params: params,
		})
	}
	for idx := 0; idx < avg; idx++ {
		fns[idx](params)
	}
	wg.Wait()
}

func P[DT any](data []DT, fn func(DT)) {
	l := len(data)
	if l < _JobUnit {
		for _, d := range data {
			fn(d)
		}
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		PushPJob(&slcJob[DT]{
			slcJobBase: slcJobBase[DT]{
				data:  data,
				start: start,
				end:   end,
				wg:    &wg,
			},
			fn: fn,
		})
	}
	for idx := 0; idx < avg; idx++ {
		fn(data[idx])
	}
	wg.Wait()
}

func PTo[DT1, DT2 any](data []DT1, fn func(DT1) DT2, complete func([]DT2)) {
	l := len(data)
	if l < _JobUnit {
		slc := make([]DT2, l)
		for i, d := range data {
			slc[i] = fn(d)
		}
		complete(slc)
		return
	}
	var wg sync.WaitGroup
	all := make([][]DT2, 0, _ParallelNum)
	end := _JobUnit << 1
	for start := _JobUnit; end < l; start = end {
		wg.Add(1)
		if end > l {
			end = l
		}
		slc := make([]DT2, 0, end-start)
		all = append(all, slc)
		PushPJob(&slcJob[DT1]{
			slcJobBase: slcJobBase[DT1]{
				data:  data,
				start: start,
				end:   end,
				wg:    &wg,
			},
			fn: func(d DT1) {
				slc = append(slc, fn(d))
			},
		})
	}
	slc := make([]DT2, _JobUnit)
	for idx := 0; idx < _JobUnit; idx++ {
		slc[idx] = fn(data[idx])
	}
	complete(slc)
	wg.Wait()
	for _, slc := range all {
		complete(slc)
	}
}

func PParams[DT any](data []DT, fn func(DT, []any), params ...any) {
	l := len(data)
	if l < _JobUnit {
		for _, d := range data {
			fn(d, params)
		}
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		PushPJob(&slcJobParams[DT]{
			slcJobBase: slcJobBase[DT]{
				data:  data,
				start: start,
				end:   end,
				wg:    &wg,
			},
			fn:     fn,
			params: params,
		})
	}
	for idx := 0; idx < _JobUnit; idx++ {
		fn(data[idx], params)
	}
	wg.Wait()
}

func PToFnLink[DT any](data []DT, fn func(DT, *ds.FnLink)) {
	l := len(data)
	if l <= _JobUnit {
		buffer := ds.NewFnLink()
		for _, d := range data {
			fn(d, buffer)
		}
		buffer.Invoke()
		buffer.Dispose()
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	buffers := make([]*ds.FnLink, 0, _ParallelNum)
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		buffer := ds.NewFnLink()
		buffers = append(buffers, buffer)
		PushPJob(&slcToFnJob[DT]{
			slcToFnJobBase: slcToFnJobBase[DT]{
				buffer: buffer,
				data:   data,
				start:  start,
				end:    end,
				wg:     &wg,
			},
			fn: fn,
		})
	}
	buffer := ds.NewFnLink()
	for idx := 0; idx < avg; idx++ {
		fn(data[idx], buffer)
	}
	buffer.Invoke()
	buffer.Dispose()
	wg.Wait()
	for _, b := range buffers {
		b.Invoke()
		b.Dispose()
	}
}

func PParamsToFnLink[DT any](data []DT, fn func(DT, []any, *ds.FnLink), params ...any) {
	l := len(data)
	if l <= _JobUnit {
		buffer := ds.NewFnLink()
		for _, d := range data {
			fn(d, params, buffer)
		}
		buffer.Invoke()
		buffer.Dispose()
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	buffers := make([]*ds.FnLink, 0, _ParallelNum)
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		buffer := ds.NewFnLink()
		buffers = append(buffers, buffer)
		PushPJob(&slcToFnJobParams[DT]{
			slcToFnJobBase: slcToFnJobBase[DT]{
				buffer: buffer,
				data:   data,
				start:  start,
				end:    end,
				wg:     &wg,
			},
			fn:     fn,
			params: params,
		})
	}
	buffer := ds.NewFnLink()
	for idx := 0; idx < avg; idx++ {
		fn(data[idx], params, buffer)
	}
	buffer.Invoke()
	buffer.Dispose()
	wg.Wait()
	for _, b := range buffers {
		b.Invoke()
		b.Dispose()
	}
}

func PToLink[InT, OutT any](data []InT, fn func(InT, *ds.Link[OutT]),
	pcr func(*ds.Link[OutT])) {
	l := len(data)
	if l <= _JobUnit {
		buffer := ds.NewLink[OutT]()
		for _, d := range data {
			fn(d, buffer)
		}
		pcr(buffer)
		return
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	buffers := make([]*ds.Link[OutT], 0, _ParallelNum)
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		buffer := ds.NewLink[OutT]()
		buffers = append(buffers, buffer)
		PushPJob(&slcToLnkJob[InT, OutT]{
			buffer: buffer,
			data:   data,
			start:  start,
			end:    end,
			fn:     fn,
			wg:     &wg,
		})
	}
	buffer := ds.NewLink[OutT]()
	for idx := 0; idx < avg; idx++ {
		fn(data[idx], buffer)
	}
	pcr(buffer)
	wg.Wait()
	for _, b := range buffers {
		pcr(b)
	}
}

func PParamsToToLink[InT, OutT any](data []InT, fn func(InT, []any, *ds.Link[OutT]),
	pcr func(*ds.Link[OutT]), params ...any) {
	l := len(data)
	if l <= _JobUnit {
		buffer := ds.NewLink[OutT]()
		for _, d := range data {
			fn(d, params, buffer)
		}
		pcr(buffer)
		return
	}
	count := l / _ParallelNum
	if l%_ParallelNum != 0 {
		count++
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	buffers := make([]*ds.Link[OutT], 0, _ParallelNum)
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		buffer := ds.NewLink[OutT]()
		buffers = append(buffers, buffer)
		PushPJob(&slcToLnkJobParams[InT, OutT]{
			buffer: buffer,
			data:   data,
			start:  start,
			end:    end,
			fn:     fn,
			wg:     &wg,
			params: params,
		})
	}
	buffer := ds.NewLink[OutT]()
	for idx := 0; idx < avg; idx++ {
		fn(data[idx], params, buffer)
	}
	pcr(buffer)
	wg.Wait()
	for _, b := range buffers {
		pcr(b)
	}
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

type IJob interface {
	Do()
}

type fnJob struct {
	fns        []util.FnAnySlc
	start, end int
	wg         *sync.WaitGroup
	params     []any
}

func (j *fnJob) Do() {
	for i := j.start; i < j.end; i++ {
		j.fns[i](j.params)
	}
	j.wg.Done()
}

type slcJobBase[DT any] struct {
	data       []DT
	start, end int
	fn         func(int, DT)
	wg         *sync.WaitGroup
}

type slcJob[DT any] struct {
	slcJobBase[DT]
	fn func(DT)
}

func (j *slcJob[DT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i])
	}
	j.wg.Done()
}

type slcJobParams[DT any] struct {
	slcJobBase[DT]
	fn     func(DT, []any)
	params []any
}

func (j *slcJobParams[DT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i], j.params)
	}
	j.wg.Done()
}

type slcToLnkJob[DT, BT any] struct {
	buffer     *ds.Link[BT]
	data       []DT
	start, end int
	fn         func(DT, *ds.Link[BT])
	wg         *sync.WaitGroup
}

func (j *slcToLnkJob[DT, BT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i], j.buffer)
	}
	j.wg.Done()
}

type slcToLnkJobParams[DT, BT any] struct {
	buffer     *ds.Link[BT]
	data       []DT
	start, end int
	fn         func(DT, []any, *ds.Link[BT])
	wg         *sync.WaitGroup
	params     []any
}

func (j *slcToLnkJobParams[DT, BT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i], j.params, j.buffer)
	}
	j.wg.Done()
}

type slcToFnJobBase[DT any] struct {
	buffer     *ds.FnLink
	data       []DT
	start, end int
	wg         *sync.WaitGroup
}

type slcToFnJob[DT any] struct {
	slcToFnJobBase[DT]
	fn func(DT, *ds.FnLink)
}

func (j *slcToFnJob[DT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i], j.buffer)
	}
	j.wg.Done()
}

type slcToFnJobParams[DT any] struct {
	slcToFnJobBase[DT]
	fn     func(DT, []any, *ds.FnLink)
	params []any
}

func (j *slcToFnJobParams[DT]) Do() {
	for i := j.start; i < j.end; i++ {
		j.fn(j.data[i], j.params, j.buffer)
	}
	j.wg.Done()
}
