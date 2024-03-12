package graph

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

type IPoint interface {
	IGraphElem
	Type() TPoint
	Node() INode
	AddLink(lnk ILink)
	GetLink(name string) (ILink, *util.Err)
	AddFilter(func(IMsg) *util.Err)
}

type TPoint string

const (
	TpNone = "none"
)

type IOut interface {
	IPoint
	Send(msg IMsg) *util.Err
}

type IIn interface {
	IPoint
	Receive(msg IMsg) *util.Err
}

func newPoint(n INode, t TPoint, name string) *point {
	return &point{
		IGraphElem: newGraphElem(n.Graph(), name),
		node:       n,
		typ:        t,
		links:      make([]ILink, 0, 1),
		nameToLink: make(map[string]ILink, 1),
	}
}

type point struct {
	IGraphElem
	node       INode
	typ        TPoint
	links      []ILink
	nameToLink map[string]ILink
	filter     *ds.FnErrLink1[IMsg]
}

func (p *point) Type() TPoint {
	return p.typ
}

func (p *point) AddLink(lnk ILink) {
	p.nameToLink[lnk.Name()] = lnk
	p.links = append(p.links, lnk)
}

func (p *point) GetLink(name string) (ILink, *util.Err) {
	lnk, ok := p.nameToLink[name]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"name":  name,
			"point": p.Name(),
			"node":  p.node.Name(),
			//"graph": p.
		})
	}
	return lnk, nil
}

func (p *point) Node() INode {
	return p.node
}

func (p *point) AddFilter(fn func(IMsg) *util.Err) {
	if p.filter == nil {
		p.filter = ds.NewFnErrLink1[IMsg]()
	}
	p.filter.Push(fn)
}

func newPIn(n INode, t TPoint, name string) IIn {
	return &in{
		point: newPoint(n, t, name),
	}
}

type in struct {
	*point
}

func (i *in) Receive(msg IMsg) *util.Err {
	if msg.Type() != i.Type() {
		return util.NewErr(util.EcType, util.M{
			"msg type": msg.Type(),
			"in":       i.Name(),
			"node":     i.node.Name(),
		})
	}
	if !i.Enabled() {
		return nil
	}
	msg.SetInPoint(i.Name())
	if i.filter != nil {
		err := i.filter.Invoke(msg)
		if err != nil {
			err.AddParam("in", i.Path())
			return err
		}
	}
	err := i.node.ProcessData(msg)
	if err != nil {
		err.AddParam("in", i.Path())
		return err
	}
	return nil
}

func newOut(n INode, t TPoint, name string) IOut {
	return &out{
		point: newPoint(n, t, name),
	}
}

type out struct {
	*point
}

func (o *out) Send(msg IMsg) *util.Err {
	if msg.Type() != o.Type() {
		return util.NewErr(util.EcType, util.M{
			"msg type": msg.Type(),
			"in type":  o.Path(),
		})
	}
	if !o.Enabled() {
		return nil
	}
	if o.filter != nil {
		err := o.filter.Invoke(msg)
		if err != nil {
			return err
		}
	}
	for _, lnk := range o.links {
		err := lnk.Transfer(msg)
		if err != nil {
			kiwi.Error(err)
		}
	}
	return nil
}
