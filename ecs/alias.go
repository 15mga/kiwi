package ecs

import "github.com/15mga/kiwi/util"

type (
	TComponent        string
	TScene            string
	TSystem           string
	TEvent            string
	FnEntity          func(*Entity)
	EntityToBool      func(*Entity) bool
	EntityToErr       func(*Entity) *util.Err
	FnEntityStr       func(*Entity, string)
	FnEntityCom       func(*Entity, IComponent)
	FnCom             func(IComponent)
	ComToBool         func(IComponent) bool
	FnEntityTCom      func(*Entity, TComponent)
	FnEntityEvent     func(*Entity, IEvent)
	FnEvent           func(IEvent)
	FnEvents          func([]IEvent)
	ToEvent           func() IEvent
	EntityToEvent     func(*Entity) IEvent
	FnScene           func(*Scene)
	SceneToErr        func(*Scene) *util.Err
	FnSceneM          func(*Scene, util.M)
	FnSceneStr        func(*Scene, string)
	FnEntityFrame     func(*Entity, *Frame)
	FnFrame           func(*Frame)
	FnEntityBuffer    func(*Entity, *Buffer)
	FnComponentBuffer func(IComponent, *Buffer)
)
