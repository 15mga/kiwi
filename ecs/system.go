package ecs

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"sync"
)

func NewSystem(t TSystem) System {
	return System{
		typ: t,
	}
}

type System struct {
	typ           TSystem
	frame         *Frame
	scene         *Scene
	wg            sync.WaitGroup
	frameBefore   *ds.FnLink
	frameAfter    *ds.FnLink
	jobNameToData map[string]jobWorker
}

func (s *System) Jobs() []string {
	jobs := make([]string, len(s.jobNameToData))
	for name := range s.jobNameToData {
		jobs = append(jobs, name)
	}
	return jobs
}

func (s *System) Frame() *Frame {
	return s.frame
}

func (s *System) Scene() *Scene {
	return s.scene
}

func (s *System) FrameBefore() *ds.FnLink {
	return s.frameBefore
}

func (s *System) FrameAfter() *ds.FnLink {
	return s.frameAfter
}

func (s *System) Type() TSystem {
	return s.typ
}

func (s *System) OnBeforeStart() {
	s.jobNameToData = make(map[string]jobWorker)
}

func (s *System) OnStart(frame *Frame) {
	s.frame = frame
	s.scene = frame.scene
	s.frameBefore = frame.before
	s.frameAfter = frame.after
}

func (s *System) OnAfterStart() {
	for name := range s.jobNameToData {
		s.frame.bindJob(name, s)
	}
}

func (s *System) OnStop() {

}

func (s *System) OnUpdate() {

}

func (s *System) PutJob(name string, data ...any) {
	w, ok := s.jobNameToData[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"name": name,
		})
		return
	}
	for _, d := range data {
		w.PushJob(d)
	}
}

func (s *System) DoJob(name string) {
	d, ok := s.jobNameToData[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"name": name,
		})
		return
	}
	d.Do()
}

func (s *System) BindJob(name string, fn util.FnAny) {
	s.jobNameToData[name] = newJobWorker(fn)
}

func (s *System) BindPJob(name string, min int, fn util.FnAny) {
	s.jobNameToData[name] = newPJobWorker(min, fn)
}

func (s *System) BindAfterPJob(name string, min int, fn FnAnyAndLink) {
	s.jobNameToData[name] = newAfterPJobWorker(min, fn)
}

func (s *System) PTagComponents(tag string, min int, fn func(IComponent)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.P[IComponent](_PJobPool, min, components, fn)
	return components, true
}

func (s *System) PComponents(components []IComponent, min int, fn func(IComponent)) {
	worker.P[IComponent](_PJobPool, min, components, fn)
}

func (s *System) PFilterTagComponents(tag string, min int, filter func(IComponent) bool, fn func([]IComponent)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PFilter[IComponent](_PFilterJobPool, min, components, filter, fn)
	return components, true
}

func (s *System) PFilterComponents(components []IComponent, min int, filter func(IComponent) bool, fn func([]IComponent)) {
	worker.PFilter[IComponent](_PFilterJobPool, min, components, filter, fn)
}

type jobWorker interface {
	PushJob(data any)
	Do()
}

type baseJobWorker struct {
	slc *ds.Array[any]
}

func (w *baseJobWorker) PushJob(data any) {
	w.slc.Add(data)
}

func newJobWorker(fn util.FnAny) *defJobWorker {
	return &defJobWorker{
		baseJobWorker: baseJobWorker{
			slc: ds.NewArray[any](64),
		},
		fn: fn,
	}
}

type defJobWorker struct {
	baseJobWorker
	fn util.FnAny
}

func (w *defJobWorker) Do() {
	for _, d := range w.slc.Values() {
		w.fn(d)
	}
	w.slc.Reset()
}

func newPJobWorker(min int, fn util.FnAny) *pJobWorker {
	return &pJobWorker{
		baseJobWorker: baseJobWorker{
			slc: ds.NewArray[any](64),
		},
		min: min,
		fn:  fn,
	}
}

type pJobWorker struct {
	baseJobWorker
	min int
	fn  util.FnAny
}

func (w *pJobWorker) Do() {
	worker.P[any](_PJobPool, w.min, w.slc.Values(), func(d any) {
		w.fn(d)
	})
	w.slc.Reset()
}

func newAfterPJobWorker(min int, fn func(any, *ds.FnLink)) *afterPJobWorker {
	return &afterPJobWorker{
		baseJobWorker: baseJobWorker{
			slc: ds.NewArray[any](64),
		},
		min: min,
		fn:  fn,
	}
}

type afterPJobWorker struct {
	baseJobWorker
	min int
	fn  func(any, *ds.FnLink)
}

func (w *afterPJobWorker) Do() {
	worker.PAfter[any](_PAfterJobPool, w.min, w.slc.Values(), func(d any, link *ds.FnLink) {
		w.fn(d, link)
	})
	w.slc.Reset()
}

var (
	_PJobPool = &sync.Pool{New: func() any {
		return &worker.PJob[IComponent]{}
	}}
	_PFilterJobPool = &sync.Pool{New: func() any {
		return &worker.PFilterJob[IComponent]{
			Array: ds.NewArray[IComponent](128),
		}
	}}
	_PAfterJobPool = &sync.Pool{New: func() any {
		return &worker.PAfterJob[IComponent]{
			Link: ds.NewFnLink(),
		}
	}}
)
