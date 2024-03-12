package ecs

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

func NewEntity(id string) *Entity {
	e := &Entity{
		id: id,
		comps: ds.NewKSet[TComponent, IComponent](1, func(component IComponent) TComponent {
			return component.Type()
		}),
	}
	return e
}

type Entity struct {
	scene *Scene
	id    string
	comps *ds.KSet[TComponent, IComponent]
}

func (e *Entity) Scene() *Scene {
	return e.scene
}

func (e *Entity) setScene(scene *Scene) {
	e.scene = scene
}

func (e *Entity) Id() string {
	return e.id
}

func (e *Entity) AddComponent(c IComponent) *util.Err {
	err := e.comps.Add(c)
	if err != nil {
		return err
	}
	c.setEntity(e)
	if e.scene == nil {
		return nil
	}
	c.Init()
	e.scene.onAddComponent(e, c)
	return nil
}

func (e *Entity) AddComponents(components ...IComponent) {
	for _, c := range components {
		ok := e.comps.AddNX(c)
		if !ok {
			continue
		}
		c.setEntity(e)
		if e.scene == nil {
			continue
		}

		c.Init()
		e.scene.onAddComponent(e, c)
	}
}

func (e *Entity) DelComponent(t TComponent) bool {
	c, ok := e.comps.Del(t)
	if !ok {
		return false
	}
	e.scene.onDelComponent(e, t)
	c.Dispose()
	return true
}

func (e *Entity) GetComponent(t TComponent) (IComponent, bool) {
	c, ok := e.comps.Get(t)
	if !ok {
		return nil, false
	}
	return c, true
}

func (e *Entity) Components() []IComponent {
	return e.comps.Values()
}

func (e *Entity) IterComponent(fn func(IComponent)) {
	e.comps.Iter(fn)
}

func (e *Entity) start() {
	for _, c := range e.comps.Values() {
		c.Start()
	}
}

func (e *Entity) Dispose() {
	for _, c := range e.comps.Values() {
		c.Dispose()
	}
	e.comps.Reset()
}
