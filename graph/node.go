package graph

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

type INode interface {
	IGraphElem
	HasIn(name string) bool
	HasOut(name string) bool
	AddIn(t TPoint, name string) *util.Err
	AddOut(t TPoint, name string) *util.Err
	GetIn(name string) (IIn, *util.Err)
	GetOut(name string) (IOut, *util.Err)
	Out(name string, m any) *util.Err
	ProcessData(msg IMsg) *util.Err
	BindFn(point string, fn MsgToErr)
}

func NewNode(g IGraph, name string) *node {
	n := &node{
		IGraphElem: newGraphElem(g, name),
		typeToIn:   make(map[string]IIn),
		typeToOut:  make(map[string]IOut),
		processor:  make(map[string]*ds.FnErrLink1[IMsg]),
	}
	return n
}

type node struct {
	IGraphElem
	typeToIn  map[string]IIn
	typeToOut map[string]IOut
	processor map[string]*ds.FnErrLink1[IMsg]
}

func (n *node) HasIn(name string) bool {
	_, ok := n.typeToIn[name]
	return ok
}

func (n *node) HasOut(name string) bool {
	_, ok := n.typeToOut[name]
	return ok
}

func (n *node) AddIn(t TPoint, name string) *util.Err {
	if _, ok := n.typeToIn[name]; ok {
		return util.NewErr(util.EcExist, util.M{
			"error": "exist in node",
			"path":  n.Path(),
			"name":  name,
		})
	}
	in := newPIn(n, t, name)
	n.typeToIn[name] = in
	plugin := n.Graph().Plugin()
	if plugin != nil {
		plugin.OnAddIn(in)
	}
	return nil
}

func (n *node) AddOut(t TPoint, name string) *util.Err {
	if _, ok := n.typeToOut[name]; ok {
		return util.NewErr(util.EcExist, util.M{
			"error": "exist out node",
			"path":  n.Path(),
			"name":  name,
		})
	}
	out := newOut(n, t, name)
	n.typeToOut[name] = out
	plugin := n.Graph().Plugin()
	if plugin != nil {
		plugin.OnAddOut(out)
	}
	return nil
}

func (n *node) GetIn(name string) (IIn, *util.Err) {
	in, ok := n.typeToIn[name]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"path":     n.Path(),
			"out node": name,
		})
	}
	return in, nil
}

func (n *node) GetOut(name string) (IOut, *util.Err) {
	out, ok := n.typeToOut[name]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"path":     n.Path(),
			"out node": name,
		})
	}
	return out, nil
}

func (n *node) Out(name string, data any) *util.Err {
	out, ok := n.typeToOut[name]
	if !ok {
		return util.NewErr(util.EcNotExist, util.M{
			"path":     n.Path(),
			"out node": name,
		})
	}
	return out.Send(&Msg{
		typ:      out.Type(),
		outNode:  n.Name(),
		outPoint: name,
		data:     data,
	})
}

func (n *node) ProcessData(msg IMsg) *util.Err {
	if !n.Enabled() {
		return nil
	}
	p, ok := n.processor[msg.InPoint()]
	if !ok {
		return util.NewErr(util.EcNotExist, util.M{
			"path":      n.Path(),
			"processor": msg.InPoint(),
		})
	}
	msg.SetInNode(n)
	err := p.Invoke(msg)
	if err != nil {
		err.AddParam("node", n.Path())
		return err
	}
	return nil
}

func (n *node) BindFn(point string, fn MsgToErr) {
	lnk, ok := n.processor[point]
	if !ok {
		lnk = ds.NewFnErrLink1[IMsg]()
		n.processor[point] = lnk
	}
	lnk.Push(fn)
}
