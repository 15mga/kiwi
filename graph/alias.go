package graph

import (
	"github.com/15mga/kiwi/util"
)

type (
	MsgToErr        func(IMsg) *util.Err
	StrLinkToBool   func(string, ILink) bool
	LinkToBool      func(ILink) bool
	NodeToAnyBool   func(INode) (any, bool)
	NodeStrToAnyErr func(INode, string) (any, bool)
	MsgToBoolErr    func(IMsg) (bool, *util.Err)
)
