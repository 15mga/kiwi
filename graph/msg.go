package graph

import (
	"github.com/15mga/kiwi/util"
)

type Msg struct {
	typ      TPoint
	outNode  string
	outPoint string
	inNode   INode
	inPoint  string
	data     any
}

func (m *Msg) Type() TPoint {
	return m.typ
}

func (m *Msg) OutNode() string {
	return m.outNode
}

func (m *Msg) OutPoint() string {
	return m.outPoint
}

func (m *Msg) InNode() INode {
	return m.inNode
}

func (m *Msg) SetInNode(inNode INode) {
	m.inNode = inNode
}

func (m *Msg) InPoint() string {
	return m.inPoint
}

func (m *Msg) SetInPoint(inPoint string) {
	m.inPoint = inPoint
}

func (m *Msg) Data() any {
	return m.data
}

func (m *Msg) ToJson() ([]byte, *util.Err) {
	return util.JsonMarshal(util.M{
		"InNode":   m.inNode,
		"InPoint":  m.inPoint,
		"OutNode":  m.outNode,
		"OutPoint": m.outPoint,
		"Data":     m.data,
	})
}

func (m *Msg) ToM() util.M {
	return util.M{
		"InNode":   m.inNode,
		"InPoint":  m.inPoint,
		"OutNode":  m.outNode,
		"OutPoint": m.outPoint,
		"Data":     m.data,
	}
}

var (
	_MsgProcessors map[string]MsgToErr
)

func InitMsgProcessor(m map[string]MsgToErr) {
	_MsgProcessors = m
}

func GetMsgProcessor(t string) MsgToErr {
	return _MsgProcessors[t]
}
