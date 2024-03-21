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
	PutJob(name JobName, data ...any)
	DoJob(name JobName)
	BindJob(name JobName, handler util.FnAnySlc)
	BindPJob(name JobName, min int, fn util.FnAnySlc)
	BindPFnJob(name JobName, min int, fn FnLinkAnySlc)
	PTagComponents(tag string, min int, fn func(IComponent)) ([]IComponent, bool)
	PTagComponentsWithParams(tag string, min int, fn func(IComponent, []any), params ...any) ([]IComponent, bool)
	PTagComponentsToFnLink(tag string, min int, fn func(IComponent, *ds.FnLink)) ([]IComponent, bool)
	PTagComponentsToFnLinkWithParams(tag string, min int, fn func(IComponent, []any, *ds.FnLink), params ...any) ([]IComponent, bool)
}

type IEvent interface {
	Type() TEvent
}

type (
	TJob         uint8
	FnLinkAnySlc func(*ds.FnLink, []any)
)

const (
	JobDef TJob = iota
	JobP
	JobPLink
)
