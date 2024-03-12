package graph

import "github.com/15mga/kiwi/util"

type IMsg interface {
	Type() TPoint
	OutNode() string
	OutPoint() string
	InNode() INode
	SetInNode(inNode INode)
	InPoint() string
	SetInPoint(inPoint string)
	Data() any
	ToJson() ([]byte, *util.Err)
	ToM() util.M
}
