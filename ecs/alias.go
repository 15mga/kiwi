package ecs

import "github.com/15mga/kiwi/util"

type (
	TComponent    string
	TScene        string
	TSystem       string
	TEvent        string
	FnEntity      func(*Entity)
	EntityToErr   func(*Entity) *util.Err
	FnEntityEvent func(*Entity, IEvent)
	FnFrame       func(*Frame)
)
