package ecs

import (
	"context"
	"github.com/15mga/kiwi/ds"
	"sync"
	"time"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
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
	c := 2048
	f := &Frame{
		option:       o,
		startTime:    now,
		nowMillSecs:  now,
		scene:        scene,
		systems:      o.systems,
		typeToSystem: make(map[TSystem]ISystem, len(o.systems)),
		jobToSystem:  make(map[string]ISystem, len(o.systems)),
		ctx:          ctx,
		ccl:          ccl,
		sign:         make(chan struct{}, 1),
		swap: &buffer{
			items: make([]*bufferData, c),
			cap:   c,
		},
		buffer: &buffer{
			items: make([]*bufferData, c),
			cap:   c,
		},
	}
	for _, system := range o.systems {
		f.typeToSystem[system.Type()] = system
	}
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
	jobToSystem  map[string]ISystem
	before       *ds.FnLink
	after        *ds.FnLink
	ctx          context.Context
	ccl          context.CancelFunc
	buffer       *buffer
	swap         *buffer
	mtx          sync.Mutex
	sign         chan struct{}
	idx          int
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
				for {
					f.mtx.Lock()
					if f.buffer.count == 0 {
						f.mtx.Unlock()
						break
					}
					f.swap, f.buffer = f.buffer, f.swap
					f.mtx.Unlock()

					f.do()
				}
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

// PushJob frame 外部使用，协程安全的
func (f *Frame) PushJob(name string, data any) {
	f.push(name, data)
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

// PutJob system内部使用，注意协程安全，需要回到主协程使用
func (f *Frame) PutJob(name string, data any) {
	system, ok := f.jobToSystem[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"system": name,
		})
		return
	}
	system.PutJob(name, data)
}

func (f *Frame) push(name string, data any) {
	f.mtx.Lock()
	f.buffer.Push(name, data)
	f.mtx.Unlock()

	select {
	case f.sign <- struct{}{}:
	default:
	}
}

func (f *Frame) bindJob(name string, system ISystem) {
	f.jobToSystem[name] = system
}

func (f *Frame) do() {
	if f.swap.count == 0 {
		return
	}
	items := f.swap.items
	for _, item := range items[f.idx:f.swap.count] {
		f.PutJob(item.name, item.data)
		item.data = nil
	}
	f.swap.count = 0
	f.idx = 0
}

type buffer struct {
	items []*bufferData
	cap   int
	count int
}

func (b *buffer) Push(name string, data any) {
	if b.count == b.cap {
		b.cap = b.count << 1
		items := make([]*bufferData, b.cap)
		copy(items, b.items)
		b.items = items
	}
	if b.items[b.count] == nil {
		b.items[b.count] = &bufferData{
			name: name,
			data: data,
		}
	} else {
		b.items[b.count].name = name
		b.items[b.count].data = data
	}
	b.count++
}

type bufferData struct {
	name string
	data any
}
