package ecs

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

func NewScene(id string, typ TScene) *Scene {
	return &Scene{
		id:   id,
		typ:  typ,
		data: util.M{},
		idToEntity: ds.NewKSet[string, *Entity](512,
			func(entity *Entity) string {
				return entity.Id()
			}),
		tagToComponents:           ds.NewSet[string, IComponent](16),
		componentTags:             make(map[IComponent]map[string]struct{}, 32),
		onBeforeAddEntityLink:     ds.NewFnErrLink1[*Entity](),
		onAfterAddEntityLink:      ds.NewFnLink1[*Entity](),
		onBeforeDisposeEntityLink: ds.NewFnErrLink1[*Entity](),
		onAfterDisposeEntityLink:  ds.NewFnLink1[*Entity](),
		onAddEntityComponentLink:  ds.NewFnLink2[*Entity, IComponent](),
		onDelEntityComponentLink:  ds.NewFnLink2[*Entity, TComponent](),
	}
}

type Scene struct {
	id                        string
	typ                       TScene
	data                      util.M
	idToEntity                *ds.KSet[string, *Entity]
	tagToComponents           *ds.KSet[string, *ds.SetItem[string, IComponent]]
	componentTags             map[IComponent]map[string]struct{} //key换成entityId_TComponent试试
	onBeforeAddEntityLink     *ds.FnErrLink1[*Entity]
	onAfterAddEntityLink      *ds.FnLink1[*Entity]
	onBeforeDisposeEntityLink *ds.FnErrLink1[*Entity]
	onAfterDisposeEntityLink  *ds.FnLink1[*Entity]
	onAddEntityComponentLink  *ds.FnLink2[*Entity, IComponent]
	onDelEntityComponentLink  *ds.FnLink2[*Entity, TComponent]
}

func (s *Scene) Id() string {
	return s.id
}

func (s *Scene) Type() TScene {
	return s.typ
}

func (s *Scene) Data() util.M {
	return s.data
}

func (s *Scene) AddEntity(e *Entity) *util.Err {
	id := e.Id()
	err := s.onBeforeAddEntityLink.Invoke(e)
	if err != nil {
		return err
	}

	if _, ok := s.idToEntity.Get(id); ok {
		return util.NewErr(util.EcExist, util.M{
			"id": id,
		})
	}

	_ = s.idToEntity.Add(e)
	e.setScene(s)
	e.start()
	for _, component := range e.Components() {
		s.TagComponent(component, string(component.Type()))
	}
	s.onAfterAddEntityLink.Invoke(e)
	return nil
}

func (s *Scene) DelEntity(id string) *util.Err {
	e, ok := s.idToEntity.Get(id)
	if !ok {
		return util.NewErr(util.EcNotExist, util.M{
			"entity id": id,
		})
	}
	err := s.onBeforeDisposeEntityLink.Invoke(e)
	if err != nil {
		return err
	}
	for _, component := range e.Components() {
		if a, ok := s.componentTags[component]; ok {
			for tag := range a {
				set, ok := s.tagToComponents.Get(tag)
				if !ok {
					continue
				}
				set.Del(component.Entity().Id())
				if set.Count() == 0 {
					s.tagToComponents.Del(tag)
				}
			}
		}
		delete(s.componentTags, component)
	}
	s.idToEntity.Del(id)
	s.onAfterDisposeEntityLink.Invoke(e)
	e.Dispose()
	return nil
}

func (s *Scene) GetEntity(id string) (*Entity, bool) {
	a, ok := s.idToEntity.Get(id)
	return a, ok
}

func (s *Scene) IterEntities(fn FnEntity) {
	s.idToEntity.Iter(fn)
}

func (s *Scene) Entities() []*Entity {
	return s.idToEntity.Values()
}

func (s *Scene) EntityCount() int {
	return s.idToEntity.Count()
}

func (s *Scene) EntityIds(ids *[]string) {
	s.idToEntity.CopyKeys(ids)
}

func (s *Scene) IsEmpty() bool {
	return s.idToEntity.Count() == 0
}

func (s *Scene) HasTagComponent(component IComponent, tag string) bool {
	a, ok := s.componentTags[component]
	if !ok {
		return false
	}
	_, ok = a[tag]
	return ok
}

func (s *Scene) TagComponent(component IComponent, tags ...string) {
	a, ok := s.componentTags[component]
	if !ok {
		a = make(map[string]struct{}, 4)
		s.componentTags[component] = a
	}
	for _, tag := range tags {
		if _, ok := a[tag]; ok {
			continue
		}
		a[tag] = struct{}{}
		ca, ok := s.tagToComponents.Get(tag)
		if !ok {
			ca = ds.NewSetItem[string, IComponent](32, tag, func(component IComponent) string {
				return component.Entity().Id()
			})
			_ = s.tagToComponents.AddNX(ca)
		}
		_ = ca.AddNX(component)
	}
}

func (s *Scene) TagEntityComponent(entityId string, t TComponent, tags ...string) {
	e, ok := s.idToEntity.Get(entityId)
	if !ok {
		return
	}
	component, ok := e.GetComponent(t)
	if !ok {
		return
	}
	s.TagComponent(component, tags...)
}

func (s *Scene) UntagComponent(component IComponent, tags ...string) {
	ca, ok := s.componentTags[component]
	if !ok {
		return
	}
	for _, tag := range tags {
		a, ok := s.tagToComponents.Get(tag)
		if !ok {
			continue
		}
		delete(ca, tag)
		a.Del(component.Entity().Id())
		if a.Count() == 0 {
			s.tagToComponents.Del(tag)
		}
	}
}

func (s *Scene) GetTagComponents(tag string) ([]IComponent, bool) {
	a, ok := s.tagToComponents.Get(tag)
	if !ok {
		return nil, false
	}
	return a.Values(), true
}

func (s *Scene) TestGetTagComponents(tag string) (*ds.SetItem[string, IComponent], bool) {
	a, ok := s.tagToComponents.Get(tag)
	if !ok {
		return nil, false
	}
	return a, true
}

func (s *Scene) ClearComponentTags(component IComponent) {
	a, ok := s.componentTags[component]
	if !ok {
		return
	}
	for tag := range a {
		set, ok := s.tagToComponents.Get(tag)
		if !ok {
			continue
		}
		set.Del(component.Entity().Id())
		if set.Count() == 0 {
			s.tagToComponents.Del(tag)
		}
	}
}

func (s *Scene) ClearTag(tag string) bool {
	a, ok := s.tagToComponents.Del(tag)
	if !ok {
		return false
	}
	for _, c := range a.Values() {
		delete(s.componentTags[c], tag)
	}
	return true
}

func (s *Scene) ClearTags(tags ...string) {
	for _, tag := range tags {
		a, ok := s.tagToComponents.Del(tag)
		if !ok {
			continue
		}
		for _, c := range a.Values() {
			delete(s.componentTags[c], tag)
		}
	}
}

func (s *Scene) TransferComponentTag(target string, origin string) {
	a, ok := s.tagToComponents.Del(origin)
	if !ok {
		return
	}
	a.ResetKey(target)
	_ = s.tagToComponents.Add(a)
	for _, component := range a.Values() {
		ca := s.componentTags[component]
		delete(ca, origin)
		ca[target] = struct{}{}
	}
}

func (s *Scene) BindBeforeAddEntity(fn EntityToErr) {
	s.onBeforeAddEntityLink.Push(fn)
}

func (s *Scene) BindAfterAddEntity(fn FnEntity) {
	s.onAfterAddEntityLink.Push(fn)
}

func (s *Scene) BindBeforeDisposeEntity(fn EntityToErr) {
	s.onBeforeDisposeEntityLink.Push(fn)
}

func (s *Scene) BindAfterDisposeEntity(fn FnEntity) {
	s.onAfterDisposeEntityLink.Push(fn)
}

func (s *Scene) BindAddComponent(fn FnEntityCom) {
	s.onAddEntityComponentLink.Push(fn)
}

func (s *Scene) BindDelComponent(fn FnEntityTCom) {
	s.onDelEntityComponentLink.Push(fn)
}

func (s *Scene) onAddComponent(e *Entity, c IComponent) {
	s.onAddEntityComponentLink.Invoke(e, c)
}

func (s *Scene) onDelComponent(e *Entity, t TComponent) {
	s.onDelEntityComponentLink.Invoke(e, t)
}

func (s *Scene) Dispose() {
	for _, entity := range s.idToEntity.Values() {
		entity.Dispose()
	}
	s.idToEntity = nil
	s.componentTags = nil
	s.tagToComponents = nil
}
