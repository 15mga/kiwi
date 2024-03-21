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

type jobData struct {
	slc    *ds.Array[*worker.Job]
	worker jobWorker
	min    int
}

type System struct {
	typ           TSystem
	frame         *Frame
	scene         *Scene
	wg            sync.WaitGroup
	frameBefore   *ds.FnLink
	frameAfter    *ds.FnLink
	jobNameToData map[worker.JobName]*jobData
}

func (s *System) Jobs() []worker.JobName {
	jobs := make([]worker.JobName, len(s.jobNameToData))
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
	s.jobNameToData = make(map[JobName]*jobData)
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

func (s *System) PutJob(name worker.JobName, data ...any) {
	d, ok := s.jobNameToData[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"name": name,
		})
		return
	}
	j := worker.SpawnJob()
	j.Name = name
	j.Data = data
	d.slc.Add(j)
}

func (s *System) DoJob(name JobName) {
	d, ok := s.jobNameToData[name]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"name": name,
		})
		return
	}
	slc := d.slc.Values()
	if len(slc) > 0 {
		d.worker.Do(d.min, slc)
		d.slc.Reset()
	}
}

func (s *System) BindJob(name JobName, fn util.FnAnySlc) {
	s.jobNameToData[name] = &jobData{
		slc: ds.NewArray[*worker.Job](8),
		worker: &defWorker{
			fn: fn,
		},
	}
}

func (s *System) BindPJob(name JobName, min int, fn util.FnAnySlc) {
	s.jobNameToData[name] = &jobData{
		slc: ds.NewArray[*worker.Job](8),
		worker: &pWorker{
			fn:  fn,
			min: min,
		},
	}
}

func (s *System) BindPFnJob(name JobName, min int, fn FnLinkAnySlc) {
	s.jobNameToData[name] = &jobData{
		slc: ds.NewArray[*worker.Job](8),
		worker: &pLinkWorker{
			fn:  fn,
			min: min,
		},
	}
}

func (s *System) PTagComponents(tag string, min int, fn func(IComponent)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.P[IComponent](min, components, fn)
	return components, true
}

func (s *System) PTagComponentsWithParams(tag string, min int, fn func(IComponent, []any), params ...any) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PParams[IComponent](min, components, fn, params...)
	return components, true
}

func (s *System) PTagComponentsToFnLink(tag string, min int, fn func(IComponent, *ds.FnLink)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PToFnLink[IComponent](min, components, fn)
	return components, true
}

func (s *System) PTagComponentsToFnLinkWithParams(tag string, min int, fn func(IComponent, []any, *ds.FnLink), params ...any) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PParamsToFnLink[IComponent](min, components, fn, params...)
	return components, true
}

func PTo[T comparable](s ISystem, tag string, min int, fn func(IComponent) (T, bool), complete func([]T)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PFilter[IComponent, T](min, components, fn, complete)
	return components, true
}

func PToLink[T any](s ISystem, tag string, min int, fn func(IComponent, *ds.Link[T]), pcr func(*ds.Link[T])) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PToLink[IComponent, T](min, components, fn, pcr)
	return components, true
}

func PToFnLink(s ISystem, tag string, min int, fn func(IComponent, *ds.FnLink)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PToFnLink[IComponent](min, components, fn)
	return components, true
}

func PParamsToToLink[T any](s ISystem, tag string, min int, fn func(IComponent, []any, *ds.Link[T]), pcr func(*ds.Link[T]), params ...any) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PParamsToToLink[IComponent, T](min, components, fn, pcr, params...)
	return components, true
}

func PParamsToFnLink(s ISystem, tag string, min int, fn func(IComponent, []any, *ds.FnLink), params ...any) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PParamsToFnLink[IComponent](min, components, fn, params...)
	return components, true
}

func PFilter[T comparable](s ISystem, tag string, min int, fn func(IComponent) (T, bool), complete func([]T)) ([]IComponent, bool) {
	components, ok := s.Scene().GetTagComponents(tag)
	if !ok {
		return nil, false
	}
	worker.PFilter[IComponent, T](min, components, fn, complete)
	return components, true
}

type jobWorker interface {
	Type() TJob
	Do(min int, jobs []*worker.Job)
}

type defWorker struct {
	fn util.FnAnySlc
}

func (w *defWorker) Type() TJob {
	return JobDef
}

func (w *defWorker) Do(min int, jobs []*worker.Job) {
	for _, j := range jobs {
		w.fn(j.Data)
		worker.RecycleJob(j)
	}
}

type pWorker struct {
	fn  util.FnAnySlc
	min int
}

func (w *pWorker) Type() TJob {
	return JobP
}

func (w *pWorker) Do(min int, jobs []*worker.Job) {
	worker.P[*worker.Job](min, jobs, func(j *worker.Job) {
		w.fn(j.Data)
		worker.RecycleJob(j)
	})
}

type pLinkWorker struct {
	fn  FnLinkAnySlc
	min int
}

func (w *pLinkWorker) Type() TJob {
	return JobPLink
}

func (w *pLinkWorker) Do(min int, jobs []*worker.Job) {
	worker.PToFnLink(min, jobs, func(j *worker.Job, link *ds.FnLink) {
		w.fn(link, j.Data)
		worker.RecycleJob(j)
	})
}
