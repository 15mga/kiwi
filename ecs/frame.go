package ecs

import (
	"context"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/worker"
	"sync"
	"time"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

const (
	cmdFrameAddSystem = "add_system"
	cmdFrameDelSystem = "del_system"
	cmdFrameJob       = "job_system"
)

type (
	frameOption struct {
		maxFrame      int64
		tickDur       time.Duration
		systems       []ISystem
		beforeDispose FnFrame
	}
	FrameOption func(o *frameOption)
)

func FrameMax(frames int64) FrameOption {
	return func(o *frameOption) {
		o.maxFrame = frames
	}
}

func FrameTickDur(dur time.Duration) FrameOption {
	return func(o *frameOption) {
		o.tickDur = dur
	}
}

func FrameSystems(systems ...ISystem) FrameOption {
	return func(o *frameOption) {
		o.systems = systems
	}
}

func FrameBeforeDispose(fn FnFrame) FrameOption {
	return func(o *frameOption) {
		o.beforeDispose = fn
	}
}

func NewFrame(scene *Scene, opts ...FrameOption) *Frame {
	o := &frameOption{
		maxFrame: 0,
		tickDur:  time.Millisecond * 100,
	}
	for _, opt := range opts {
		opt(o)
	}
	ctx, ccl := context.WithCancel(util.Ctx())
	now := time.Now().UnixMilli()
	f := &Frame{
		option:       o,
		startTime:    now,
		nowMillSecs:  now,
		scene:        scene,
		systems:      o.systems,
		typeToSystem: make(map[TSystem]ISystem, len(o.systems)),
		jobToSystem:  make(map[worker.JobName]ISystem, len(o.systems)),
		sign:         make(chan struct{}, 1),
		ctx:          ctx,
		ccl:          ccl,
	}
	for _, system := range o.systems {
		f.typeToSystem[system.Type()] = system
	}
	f.cmdBuffer = newBuffer()
	f.before = ds.NewFnLink()
	f.after = ds.NewFnLink()
	return f
}

type Frame struct {
	option       *frameOption
	currFrame    int64
	maxFrame     int64
	totalFrameMs int64
	maxMs        int64
	deltaMs      int64
	startTime    int64
	nowMillSecs  int64
	scene        *Scene
	systems      []ISystem
	typeToSystem map[TSystem]ISystem
	jobToSystem  map[worker.JobName]ISystem
	cmdBuffer    *Buffer
	before       *ds.FnLink
	after        *ds.FnLink
	mtx          sync.Mutex
	sign         chan struct{}
	head         *job
	tail         *job
	ctx          context.Context
	ccl          context.CancelFunc
}

func (f *Frame) Num() int64 {
	return f.currFrame
}

func (f *Frame) DeltaMillSec() int64 {
	return f.deltaMs
}

func (f *Frame) StartTime() int64 {
	return f.startTime
}

func (f *Frame) NowMillSecs() int64 {
	return f.nowMillSecs
}

func (f *Frame) Scene() *Scene {
	return f.scene
}

// Before 每帧末尾调用
func (f *Frame) Before() *ds.FnLink {
	return f.before
}

// After 每帧末尾调用
func (f *Frame) After() *ds.FnLink {
	return f.after
}

// GetSystem 注意协程安全
func (f *Frame) GetSystem(typ TSystem) (ISystem, bool) {
	sys, ok := f.typeToSystem[typ]
	return sys, ok
}

func (f *Frame) Start() {
	completeCh := kiwi.BeforeExitCh("stop frame")
	go func() {
		defer func() {
			if f.option.beforeDispose != nil {
				f.option.beforeDispose(f)
			}

			for _, system := range f.systems {
				kiwi.Info("stop system", util.M{
					"type": system.Type(),
				})
				system.OnStop()
			}

			kiwi.Info("dispose scene", util.M{
				"scene type": f.scene.typ,
				"scene id":   f.scene.id,
			})
			f.scene.Dispose()

			if f.currFrame > 0 {
				kiwi.Info("frames", util.M{
					"total":   f.totalFrameMs,
					"average": f.totalFrameMs / f.currFrame,
					"max":     f.maxMs,
					"frames":  f.currFrame,
				})
			}
			close(completeCh)
		}()

		for _, system := range f.systems {
			system.OnBeforeStart()
			system.OnStart(f)
			system.OnAfterStart()
			kiwi.Info("start system", util.M{
				"type": system.Type(),
			})
		}

		ctx := f.ctx
		ticker := time.NewTicker(f.option.tickDur)
		for {
			select {
			case <-ctx.Done():
				kiwi.Debug("ctx done", nil)
				ticker.Stop()
				return
			case <-ticker.C:
				f.tick()
			case <-f.sign:
				f.do()
			}
		}
	}()
}

func (f *Frame) Stop() {
	f.ccl()
}

func (f *Frame) tick() {
	f.currFrame++
	now := time.Now().UnixMilli()
	ms := now - f.nowMillSecs
	f.nowMillSecs = now
	f.before.InvokeAndReset()
	for _, s := range f.systems {
		s.OnUpdate()
	}
	f.after.InvokeAndReset()
	f.deltaMs = ms
	frameDur := time.Now().UnixMilli() - now
	//kiwi.Debug("frame", util.M{
	//	"curr": f.currFrame,
	//	"dur":  frameDur,
	//})
	f.totalFrameMs += frameDur
	if ms > f.maxMs {
		f.maxMs = ms
	}
}

func (f *Frame) AddSystem(system ISystem, before TSystem) {
	f.push(cmdFrameAddSystem, system, before)
}

func (f *Frame) DelSystem(t TSystem) {
	f.push(cmdFrameDelSystem, t)
}

// PushJob frame 外部使用，协程安全的
func (f *Frame) PushJob(name JobName, data ...any) {
	f.push(cmdFrameJob, name, data)
}

func (f *Frame) AfterClearTags(tags ...string) {
	f.after.Push(func() {
		f.Scene().ClearTags(tags...)
	})
}

func (f *Frame) onAddSystem(data []any) {
	system, before := util.SplitSlc2[ISystem, TSystem](data)
	t := system.Type()
	idx := len(f.systems)
	for i, s := range f.systems {
		if s.Type() == t {
			kiwi.Error2(util.EcExist, util.M{
				"system": t,
			})
			return
		}
		if s.Type() == before {
			idx = i
			break
		}
	}
	system.OnStart(f)
	system.OnAfterStart()
	kiwi.Info("start system", util.M{
		"type": system.Type(),
	})
	f.systems = append(append(f.systems[:idx], system), f.systems[idx:]...)
	f.typeToSystem[system.Type()] = system
}

func (f *Frame) onDelSystem(data []any) {
	t := data[0].(TSystem)
	for i, s := range f.systems {
		if s.Type() == t {
			s.OnStop()
			kiwi.Info("stop system", util.M{
				"type": s.Type(),
			})
			f.systems = append(f.systems[:i], f.systems[i+1:]...)
			delete(f.typeToSystem, t)
			break
		}
	}
}

func (f *Frame) onJobSystem(data []any) {
	name, params := util.SplitSlc2[string, []any](data)
	f.PutJob(name, params...)
}

// PutJob system内部使用，注意协程安全，需要回到主协程使用
func (f *Frame) PutJob(name JobName, data ...any) {
	system, ok := f.jobToSystem[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"system": name,
		})
		return
	}
	system.PutJob(name, data...)
}

func (f *Frame) push(cmd JobName, data ...any) {
	j := _JobPool.Get().(*job)
	j.Name = cmd
	j.Data = data

	f.mtx.Lock()
	if f.head != nil {
		f.tail.next = j
	} else {
		f.head = j
	}
	f.tail = j
	f.mtx.Unlock()

	select {
	case f.sign <- struct{}{}:
	default:
	}
}

func (f *Frame) bindJob(name worker.JobName, system ISystem) {
	f.jobToSystem[name] = system
}

func (f *Frame) do() {
	f.mtx.Lock()
	head := f.head
	f.head = nil
	f.tail = nil
	f.mtx.Unlock()

	if head == nil {
		return
	}

	for j := head; j != nil; {
		switch j.Name {
		case cmdFrameAddSystem:
			f.onAddSystem(j.Data)
		case cmdFrameDelSystem:
			f.onDelSystem(j.Data)
		case cmdFrameJob:
			f.onJobSystem(j.Data)
		}
		next := j.next
		j.Data = nil
		j.next = nil
		_JobPool.Put(j)
		j = next
	}
}
