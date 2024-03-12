package graph

import (
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

type IGraphElem interface {
	Name() string
	Path() string
	Comment() string
	SetComment(c string)
	Data() util.M
	SetData(data util.M)
	Start() *util.Err
	Enable() *util.Err               //即使已可用,也会重复执行
	Disable() *util.Err              //即使不可用,也会重复执行
	SetEnable(enable bool) *util.Err //已经是该状态不会重复执行
	Enabled() bool
	AddBeforeEnable(fn util.BoolToErr) //添加节点切换是否可用之前调用
	DelBeforeEnable(fn util.BoolToErr) //移除节点切换是否可用之前调用
	AddAfterEnable(fn util.FnBool)     //添加节点切换是否可用之后调用
	DelAfterEnable(fn util.FnBool)     //移除节点切换是否可用之后调用
	Graph() IGraph
	RootGraph() IGraph
}

func newGraphElem(g IGraph, name string) IGraphElem {
	e := &graphElem{
		name:         name,
		enabled:      true,
		beforeEnable: ds.NewFnErrLink1[bool](),
		afterEnable:  ds.NewFnLink1[bool](),
		g:            g,
		data:         util.M{},
	}
	sg, ok := g.(ISubGraph)
	if ok {
		e.path = sg.Path() + "." + name
	} else {
		e.path = name
	}
	pg, ok := g.(ISubGraph)
	if ok {
		e.rg = pg.RootGraph()
	} else {
		e.rg = e.Graph()
	}
	return e
}

type graphElem struct {
	name         string
	comment      string
	path         string
	data         util.M
	enabled      bool
	beforeEnable *ds.FnErrLink1[bool]
	afterEnable  *ds.FnLink1[bool]
	g            IGraph
	rg           IGraph
}

func (e *graphElem) Name() string {
	return e.name
}

func (e *graphElem) Path() string {
	return e.path
}

func (e *graphElem) Comment() string {
	return e.comment
}

func (e *graphElem) SetComment(comment string) {
	e.comment = comment
}

func (e *graphElem) Data() util.M {
	return e.data
}

func (e *graphElem) SetData(data util.M) {
	e.data = data
}

func (e *graphElem) Start() *util.Err {
	if e.enabled {
		err := e.Enable()
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *graphElem) Enable() *util.Err {
	if e.beforeEnable != nil {
		err := e.beforeEnable.Invoke(true)
		if err != nil {
			return err
		}
	}
	e.enabled = true
	e.afterEnable.Invoke(true)
	return nil
}

func (e *graphElem) Disable() *util.Err {
	if e.beforeEnable != nil {
		err := e.beforeEnable.Invoke(false)
		if err != nil {
			return err
		}
	}
	e.enabled = false
	e.afterEnable.Invoke(false)
	return nil
}

func (e *graphElem) SetEnable(enable bool) *util.Err {
	if e.enabled == enable {
		return nil
	}
	if enable {
		return e.Enable()
	}

	return e.Disable()
}

func (e *graphElem) AddBeforeEnable(fn util.BoolToErr) {
	if e.beforeEnable == nil {
		e.beforeEnable = ds.NewFnErrLink1[bool]()
	}
	e.beforeEnable.Push(fn)
}

func (e *graphElem) DelBeforeEnable(fn util.BoolToErr) {
	if e.beforeEnable == nil {
		return
	}
	e.beforeEnable.Del(fn)
}

func (e *graphElem) AddAfterEnable(fn util.FnBool) {
	if e.afterEnable == nil {
		e.afterEnable = ds.NewFnLink1[bool]()
	}
	e.afterEnable.Push(fn)
}

func (e *graphElem) DelAfterEnable(fn util.FnBool) {
	if e.afterEnable == nil {
		return
	}
	e.afterEnable.Del(fn)
}

func (e *graphElem) Enabled() bool {
	return e.enabled
}

func (e *graphElem) RootGraph() IGraph {
	return e.rg
}

func (e *graphElem) Graph() IGraph {
	return e.g
}
