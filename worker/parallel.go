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
	_WorkerNum        int
	_WorkerNum32      uint32
	_JobParallelCount int
)

func init() {
	_ParallelNum = runtime.NumCPU()
	if _ParallelNum < 8 {
		_ParallelNum = 8
	}
	_WorkerNum = _ParallelNum - 1
	_WorkerNum32 = uint32(_WorkerNum)
	_JobParallelCount = _JobUnit * _WorkerNum
}

const (
	_JobUnit = 128
)

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

func getAvgCount(l int) int {
	if l < _JobParallelCount {
		return _JobUnit
	}
	num := _ParallelNum
	count := l / num
	if l%num != 0 {
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

func PFilter[DT1 any, DT2 comparable](data []DT1, fn func(DT1) (DT2, bool), complete func([]DT2)) {
	l := len(data)
	if l < _JobUnit {
		slc := make([]DT2, 0, l)
		for _, d := range data {
			item, ok := fn(d)
			if ok {
				slc = append(slc, item)
			}
		}
		if len(slc) > 0 {
			complete(slc)
		}
		return
	}
	var wg sync.WaitGroup
	all := make([]*ds.Array[DT2], 0, _WorkerNum)
	avg := getAvgCount(l)
	var end int
	for start := avg; end < l; start += avg {
		end = start + avg
		if end > l {
			end = l
		}
		wg.Add(1)
		arr := ds.NewArray[DT2](end - start)
		all = append(all, arr)
		PushPJob(&slcJob[DT1]{
			slcJobBase: slcJobBase[DT1]{
				data:  data,
				start: start,
				end:   end,
				wg:    &wg,
			},
			fn: func(d DT1) {
				item, ok := fn(d)
				if ok {
					arr.Add(item)
				}
			},
		})
	}
	slc := make([]DT2, 0, avg)
	for idx := 0; idx < avg; idx++ {
		item, ok := fn(data[idx])
		if ok {
			slc = append(slc, item)
		}
	}
	if len(slc) > 0 {
		complete(slc)
	}
	wg.Wait()
	for _, arr := range all {
		if arr.Count() > 0 {
			complete(arr.Values())
		}
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
	for idx := 0; idx < avg; idx++ {
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
	buffers := make([]*ds.FnLink, 0, _WorkerNum)
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
	buffers := make([]*ds.FnLink, 0, _WorkerNum)
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
	buffers := make([]*ds.Link[OutT], 0, _WorkerNum)
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
	count := l / _WorkerNum
	if l%_WorkerNum != 0 {
		count++
	}
	var wg sync.WaitGroup
	avg := getAvgCount(l)
	var end int
	buffers := make([]*ds.Link[OutT], 0, _WorkerNum)
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
