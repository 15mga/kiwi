package worker

import (
	"github.com/15mga/kiwi/util"
	"time"

	"github.com/15mga/kiwi/ds"
)

const (
	cmdActiveCheck   = "check"
	cmdActiveDispose = "dispose"
	cmdActivePush    = "push"
)

type (
	ActiveOption func(o *activeOption)
	activeOption struct {
		tickSecs int64
		cap      int
	}
)

// ActiveTickSecs 设置活跃的检测间隔时间
func ActiveTickSecs(seconds int64) ActiveOption {
	return func(opt *activeOption) {
		opt.tickSecs = seconds
	}
}

// ActiveCap 设置活跃初始容量
func ActiveCap(cap int) ActiveOption {
	return func(opt *activeOption) {
		opt.cap = cap
	}
}

type activeData struct {
	ts int64
	id string
}

var (
	_Active *active
)

func Active() *active {
	return _Active
}

func InitActive(opts ...ActiveOption) {
	if _Active != nil {
		return
	}
	opt := &activeOption{
		tickSecs: 32,
		cap:      4096,
	}
	for _, o := range opts {
		o(opt)
	}
	_Active = &active{
		option:  opt,
		closeCh: make(chan struct{}, 1),
		activeWorkers: ds.NewKSet[string, *activeWorker](opt.cap, func(a *activeWorker) string {
			return a.id
		}),
		activeTimeStamp: ds.NewKSet[string, *activeData](opt.cap, func(data *activeData) string {
			return data.id
		}),
		activeStopSeconds: opt.tickSecs << 1,
	}
	_Active.worker = NewJobWorker(_Active.process)
	go func() {
		ticker := time.NewTicker(time.Duration(_Active.option.tickSecs) * time.Second)
		defer func() {
			ticker.Stop()
			close(_Active.closeCh)
		}()
		for {
			select {
			case <-ticker.C:
				_Active.worker.Push(cmdActiveCheck)
			case <-_Active.closeCh:
				return
			}
		}
	}()
	_Active.worker.Start()
}

type active struct {
	option            *activeOption
	worker            *JobWorker
	closeCh           chan struct{}
	activeWorkers     *ds.KSet[string, *activeWorker]
	activeStopSeconds int64
	activeTimeStamp   *ds.KSet[string, *activeData]
}

func (a *active) Dispose() {
	a.worker.Push(cmdActiveDispose)
	a.worker.Dispose()
}

func (a *active) Push(id string, fn util.FnAnySlc, params ...any) {
	a.worker.Push(cmdActivePush, id, fn, params)
}

func (a *active) process(job *Job) {
	switch job.Name {
	case cmdActiveCheck:
		now := time.Now().Unix()
		//移除不活跃的协程
		a.activeTimeStamp.TestDel(func(id string, item *activeData) (del bool, brk bool) {
			if now-item.ts < a.activeStopSeconds {
				return
			}
			obj, _ := a.activeWorkers.Del(id)
			obj.Dispose()
			del = true
			return
		})
	case cmdActiveDispose:
		a.activeWorkers.Iter(func(item *activeWorker) {
			item.Dispose()
		})
	case cmdActivePush:
		id, fn, params := util.SplitSlc3[string, util.FnAnySlc, []any](job.Data)
		worker, ok := a.activeWorkers.Get(id)
		now := time.Now().Unix()
		if ok {
			d, _ := a.activeTimeStamp.Get(id)
			d.ts = now
		} else {
			worker = newActiveWorker(id)
			worker.Start()
			a.activeWorkers.Set(worker)
			_ = a.activeTimeStamp.Add(&activeData{
				ts: now,
				id: id,
			})
		}
		worker.Push(fn, params...)
	}
}

func newActiveWorker(id string) *activeWorker {
	a := &activeWorker{
		FnWorker: NewFnWorker(),
		id:       id,
	}
	return a
}

type activeWorker struct {
	*FnWorker
	id string
}
