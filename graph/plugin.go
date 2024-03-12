package graph

type IPlugin interface {
	OnStart(g IGraph)
	OnAddNode(g IGraph, nd INode)
	OnAddSubGraph(g IGraph, sg ISubGraph)
	OnAddLink(g IGraph, lnk ILink)
	OnAddIn(in IIn)
	OnAddOut(out IOut)
}
