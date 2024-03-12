package graph

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

type ILink interface {
	IGraphElem
	In() IIn
	Out() IOut
	Transfer(msg IMsg) *util.Err
	AddFilter(fn func(IMsg) *util.Err)
}

func newLink(g IGraph, name string, out IOut, in IIn) ILink {
	return &link{
		IGraphElem: newGraphElem(g, name),
		out:        out,
		in:         in,
	}
}

type link struct {
	IGraphElem
	in     IIn
	out    IOut
	filter *ds.FnErrLink1[IMsg]
}

func (l *link) In() IIn {
	return l.in
}

func (l *link) Out() IOut {
	return l.out
}

func (l *link) Transfer(msg IMsg) *util.Err {
	if !l.Enabled() {
		return nil
	}
	if l.filter != nil {
		err := l.filter.Invoke(msg)
		if err != nil {
			err.AddParam("link", l.Path())
			return err
		}
	}
	return l.in.Receive(msg)
}

func (l *link) AddFilter(fn func(IMsg) *util.Err) {
	if l.filter == nil {
		l.filter = ds.NewFnErrLink1[IMsg]()
	}
	l.filter.Push(fn)
}
