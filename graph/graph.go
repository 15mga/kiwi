package graph

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

type IGraph interface {
	Name() string
	Data() util.M
	SetData(m util.M)
	Plugin() IPlugin
	SetPlugin(plugin IPlugin)
	Start() *util.Err
	GetNode(name string) INode
	GetNodeByPath(names ...string) (INode, *util.Err)
	FindNode(name string) (INode, bool)
	FindNodes(name string, nodes *[]INode)
	IterNode(fn func(INode))
	AnyNode(fn func(INode) bool) bool
	Link(outNode, outPoint, inNode, inPoint string) (ILink, *util.Err)
	IterLink(fn func(ILink))
	AnyLink(fn func(ILink) bool) bool
	AddSubGraph(name string) (ISubGraph, *util.Err)
	GetSubGraph(graph string) (ISubGraph, *util.Err)
	IterSubGraph(fn func(ISubGraph))
	AnySubGraph(fn func(ISubGraph) bool) bool
}

type (
	option struct {
		plugin IPlugin
	}
	Option func(*option)
)

func Plugin(plugin IPlugin) Option {
	return func(o *option) {
		o.plugin = plugin
	}
}

func NewGraph(name string, opts ...Option) IGraph {
	opt := &option{}
	for _, o := range opts {
		o(opt)
	}
	g := &graph{
		option:      opt,
		name:        name,
		data:        util.M{},
		nameToNode:  make(map[string]INode),
		nameToGraph: make(map[string]ISubGraph),
		nameToLink:  make(map[string]ILink),
	}
	return g
}

type graph struct {
	option      *option
	name        string
	data        util.M
	nameToNode  map[string]INode
	nameToGraph map[string]ISubGraph
	nameToLink  map[string]ILink
}

func (g *graph) Name() string {
	return g.name
}

func (g *graph) Data() util.M {
	return g.data
}

func (g *graph) SetData(data util.M) {
	g.data = data
}

func (g *graph) Plugin() IPlugin {
	return g.option.plugin
}

func (g *graph) SetPlugin(plugin IPlugin) {
	g.option.plugin = plugin
}

func (g *graph) Start() *util.Err {
	for _, nd := range g.nameToNode {
		err := nd.Start()
		if err != nil {
			return err
		}
	}
	for _, lnk := range g.nameToLink {
		err := lnk.Start()
		if err != nil {
			return err
		}
	}
	if g.option.plugin != nil {
		g.option.plugin.OnStart(g)
	}
	return nil
}

func (g *graph) GetNode(name string) INode {
	n, ok := g.nameToNode[name]
	if ok {
		return n
	}

	nd := NewNode(g, name)
	g.nameToNode[name] = nd
	if g.option.plugin != nil {
		g.option.plugin.OnAddNode(g, nd)
	}
	return nd
}

func (g *graph) GetNodeByPath(names ...string) (INode, *util.Err) {
	switch len(names) {
	case 0:
		return nil, util.NewErr(util.EcLengthErr, nil)
	case 1:
		return g.GetNode(names[0]), nil
	default:
		sgn := names[0]
		sg, ok := g.nameToGraph[sgn]
		if !ok {
			return nil, util.NewErr(util.EcNotExist, util.M{
				"subgraph name": sgn,
				"graph":         g.Name(),
			})
		}
		return sg.GetNodeByPath(names[1:]...)
	}
}

func (g *graph) FindNode(name string) (INode, bool) {
	for nm, nd := range g.nameToNode {
		if nm == name {
			return nd, true
		}
	}
	return nil, false
}

func (g *graph) FindNodes(name string, nodes *[]INode) {
	for nm, nd := range g.nameToNode {
		if nm == name {
			*nodes = append(*nodes, nd)
		}
	}
	for _, sg := range g.nameToGraph {
		sg.FindNodes(name, nodes)
	}
}

func (g *graph) IterNode(fn func(INode)) {
	for _, nd := range g.nameToNode {
		fn(nd)
	}
}

func (g *graph) AnyNode(fn func(INode) bool) bool {
	for _, nd := range g.nameToNode {
		if fn(nd) {
			return true
		}
	}
	return false
}

func (g *graph) Link(outNode, outPoint, inNode, inPoint string) (ILink, *util.Err) {
	name := LinkName(outNode, outPoint, inNode, inPoint)
	_, ok := g.nameToLink[name]
	if ok {
		return nil, util.NewErr(util.EcExist, util.M{
			"name":  name,
			"graph": g.Name(),
		})
	}

	on := g.GetNode(outNode)
	op, err := on.GetOut(outPoint)
	if err != nil {
		err.AddParam("graph", g.Name())
		return nil, err
	}

	in := g.GetNode(inNode)
	ip, err := in.GetIn(inPoint)
	if err != nil {
		err.AddParam("graph", g.Name())
		return nil, err
	}

	if ip.Type() != op.Type() {
		return nil, util.NewErr(util.EcParamsErr, util.M{
			"name":          name,
			"graph":         g.Name(),
			"in node type":  ip.Type(),
			"out node type": op.Type(),
		})
	}

	lnk := newLink(g, name, op, ip)
	op.AddLink(lnk)
	ip.AddLink(lnk)
	g.nameToLink[name] = lnk
	if g.option.plugin != nil {
		g.option.plugin.OnAddLink(g, lnk)
	}
	return lnk, nil
}

func (g *graph) IterLink(fn func(ILink)) {
	for _, lnk := range g.nameToLink {
		fn(lnk)
	}
}

func (g *graph) AnyLink(fn func(ILink) bool) bool {
	for _, lnk := range g.nameToLink {
		if fn(lnk) {
			return true
		}
	}
	return false
}

func (g *graph) AddSubGraph(name string) (ISubGraph, *util.Err) {
	_, ok := g.nameToNode[name]
	if ok {
		return nil, util.NewErr(util.EcExist, util.M{
			"name":  name,
			"graph": g.Name(),
		})
	}
	sg := newSubGraph(g, name)
	g.nameToGraph[name] = sg
	g.nameToNode[name] = sg
	if g.option.plugin != nil {
		g.option.plugin.OnAddNode(g, sg)
		g.option.plugin.OnAddSubGraph(g, sg)
	}
	return sg, nil
}

func (g *graph) GetSubGraph(name string) (ISubGraph, *util.Err) {
	sg, ok := g.nameToGraph[name]
	if !ok {
		return nil, util.NewErr(util.EcNotExist, util.M{
			"name":  name,
			"graph": g.Name(),
		})
	}
	return sg, nil
}

func (g *graph) IterSubGraph(fn func(ISubGraph)) {
	for _, sg := range g.nameToGraph {
		fn(sg)
	}
}

func (g *graph) AnySubGraph(fn func(ISubGraph) bool) bool {
	for _, sg := range g.nameToGraph {
		if fn(sg) {
			return true
		}
	}
	return false
}

func NewGraphWithConf(conf Conf, opts ...Option) IGraph {
	g := NewGraph(conf.Name, opts...)
	for _, nc := range conf.Nodes {
		_, err := AddNodeWithConf(g, nc)
		if err != nil {
			kiwi.Error(err)
		}
	}
	for _, gc := range conf.SubGraphs {
		_, err := AddSubGraphWithConf(g, gc)
		if err != nil {
			kiwi.Error(err)
		}
	}
	for _, lc := range conf.Links {
		_, err := LinkWithConf(g, lc)
		if err != nil {
			kiwi.Error(err)
		}
	}
	return g
}

func AddNodeWithConf(g IGraph, conf NodeConf) (INode, *util.Err) {
	n := g.GetNode(conf.Name)
	n.SetComment(conf.Comment)
	if conf.M != nil {
		n.SetData(conf.M)
	} else {
		n.SetData(util.M{})
	}

	for _, ip := range conf.Ins {
		err := n.AddIn(ip.Type, ip.Name)
		if err != nil {
			kiwi.Error(err)
		}
	}
	for _, op := range conf.Outs {
		err := n.AddOut(op.Type, op.Name)
		if err != nil {
			kiwi.Error(err)
		}
	}

	for pnt, name := range conf.PointToProcessor {
		pcr, ok := _MsgProcessors[name]
		if !ok {
			kiwi.Error2(util.EcNotExist, util.M{
				"name": name,
			})
			continue
		}
		n.BindFn(pnt, pcr)
	}
	return n, nil
}

func AddSubGraphWithConf(g IGraph, conf SubConf) (ISubGraph, *util.Err) {
	sg, err := g.AddSubGraph(conf.Name)
	if err != nil {
		err.AddParam("graph", g.Name())
		return nil, err
	}

	for _, nc := range conf.Nodes {
		_, err = AddNodeWithConf(sg, nc)
		if err != nil {
			err.AddParam("graph", g.Name())
			kiwi.Error(err)
		}
	}

	for _, gc := range conf.SubGraphs {
		_, err = AddSubGraphWithConf(sg, gc)
		if err != nil {
			err.AddParam("graph", g.Name())
			kiwi.Error(err)
		}
	}
	_ = sg.SetInNode(conf.In)
	_ = sg.SetOutNode(conf.Out)

	for _, lc := range conf.Links {
		_, err = LinkWithConf(sg, lc)
		if err != nil {
			err.AddParam("graph", g.Name())
			kiwi.Error(err)
		}
	}
	return sg, nil
}

func LinkWithConf(g IGraph, conf LinkConf) (ILink, *util.Err) {
	return g.Link(conf.OutNode, conf.OutPoint, conf.InNode, conf.InPoint)
}
