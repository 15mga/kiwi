package graph

import (
	"github.com/15mga/kiwi/util"
)

type ISubGraph interface {
	IGraph
	INode
	InNode() INode
	SetInNode(name string) *util.Err
	OutNode() INode
	SetOutNode(name string) *util.Err
}

func newSubGraph(parent IGraph, name string) *subGraph {
	sg := &subGraph{
		IGraph:    NewGraph(name, Plugin(parent.Plugin())),
		graphElem: newGraphElem(parent, name),
	}
	return sg
}

type subGraph struct {
	IGraph
	graphElem IGraphElem
	inNode    INode
	outNode   INode
}

func (g *subGraph) Path() string {
	return g.graphElem.Path()
}

func (g *subGraph) Enable() *util.Err {
	return g.graphElem.Enable()
}

func (g *subGraph) Disable() *util.Err {
	return g.graphElem.Disable()
}

func (g *subGraph) SetEnable(enable bool) *util.Err {
	return g.graphElem.SetEnable(enable)
}

func (g *subGraph) Enabled() bool {
	return g.graphElem.Enabled()
}

func (g *subGraph) Graph() IGraph {
	return g.graphElem.Graph()
}

func (g *subGraph) RootGraph() IGraph {
	return g.graphElem.RootGraph()
}

func (g *subGraph) Comment() string {
	return g.graphElem.Comment()
}

func (g *subGraph) SetComment(c string) {
	g.graphElem.SetComment(c)
}

func (g *subGraph) Start() *util.Err {
	err := g.graphElem.Start()
	if err != nil {
		return err
	}
	err = g.IGraph.Start()
	if err != nil {
		return err
	}
	return nil
}

func (g *subGraph) AddBeforeEnable(fn util.BoolToErr) {
	g.graphElem.AddBeforeEnable(fn)
}

func (g *subGraph) DelBeforeEnable(fn util.BoolToErr) {
	g.graphElem.DelBeforeEnable(fn)
}

func (g *subGraph) AddAfterEnable(fn util.FnBool) {
	g.graphElem.AddAfterEnable(fn)
}

func (g *subGraph) DelAfterEnable(fn util.FnBool) {
	g.graphElem.DelAfterEnable(fn)
}

func (g *subGraph) AddIn(t TPoint, name string) *util.Err {
	if g.inNode == nil {
		return util.NewErr(util.EcNotExist, util.M{
			"error":    "not exist in node",
			"subGraph": g.Path(),
		})
	}
	return g.inNode.AddIn(t, name)
}

func (g *subGraph) AddOut(t TPoint, name string) *util.Err {
	if g.outNode == nil {
		return util.NewErr(util.EcNotExist, util.M{
			"error":    "not exist out node",
			"subGraph": g.Path(),
		})
	}
	return g.outNode.AddOut(t, name)
}

func (g *subGraph) InNode() INode {
	return g.inNode
}

func (g *subGraph) SetInNode(name string) *util.Err {
	if g.inNode != nil {
		return util.NewErr(util.EcExist, util.M{
			"error":     "in node not nil",
			"sub graph": g.Name(),
		})
	}
	nd, err := g.GetNodeByPath(name)
	if err != nil {
		return err
	}
	g.inNode = nd
	return nil
}

func (g *subGraph) OutNode() INode {
	return g.outNode
}

func (g *subGraph) SetOutNode(name string) *util.Err {
	if g.outNode != nil {
		return util.NewErr(util.EcExist, util.M{
			"error":     "out node not nil",
			"sub graph": g.Name(),
		})
	}
	nd, err := g.GetNodeByPath(name)
	if err != nil {
		return err
	}
	g.outNode = nd
	return nil
}

func (g *subGraph) GetIn(name string) (IIn, *util.Err) {
	if g.inNode == nil {
		return nil, util.NewErr(util.EcNil, util.M{
			"error":     "not set in node",
			"sub graph": g.Name(),
		})
	}
	return g.inNode.GetIn(name)
}

func (g *subGraph) GetOut(name string) (IOut, *util.Err) {
	if g.outNode == nil {
		return nil, util.NewErr(util.EcNil, util.M{
			"error":     "not set out node",
			"sub graph": g.Name(),
		})
	}
	return g.outNode.GetOut(name)
}

func (g *subGraph) Out(name string, m any) *util.Err {
	return g.outNode.Out(name, m)
}

func (g *subGraph) ProcessData(msg IMsg) *util.Err {
	return g.inNode.ProcessData(msg)
}

func (g *subGraph) HasIn(tag string) bool {
	return g.inNode.HasIn(tag)
}

func (g *subGraph) HasOut(tag string) bool {
	return g.outNode.HasOut(tag)
}

func (g *subGraph) BindFn(name string, fn MsgToErr) {
	g.inNode.BindFn(name, fn)
}
