package ecs

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

type IComponent interface {
	Entity() *Entity
	setEntity(entity *Entity)
	Type() TComponent
	// Init 添加到Entity时调用
	Init()
	// Start Entity添加到Scene时调用
	Start()
	Dispose()
}

type ISystem interface {
	Type() TSystem
	Frame() *Frame
	Scene() *Scene
	FrameBefore() *ds.FnLink
	FrameAfter() *ds.FnLink
	OnBeforeStart()
	OnStart(frame *Frame)
	OnAfterStart()
	OnStop()
	OnUpdate()
	PutJob(name string, data ...any)
	DoJob(name string)
	BindJob(name string, handler util.FnAny)
	BindPJob(name string, min int, fn util.FnAny)
	BindAfterPJob(name string, min int, fn FnAnyAndLink)
	PTagComponents(tag string, min int, fn func(IComponent)) ([]IComponent, bool)
	PComponents(components []IComponent, min int, fn func(IComponent))
	PFilterTagComponents(tag string, min int, filter func(IComponent) bool, fn func([]IComponent)) ([]IComponent, bool)
	PFilterComponents(components []IComponent, min int, filter func(IComponent) bool, fn func([]IComponent))
}

type IEvent interface {
	Type() TEvent
}

type (
	FnAnyAndLink func(any, *ds.FnLink)
)
