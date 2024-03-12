package graph

import (
	"github.com/15mga/kiwi/util"
)

type PointConf struct {
	Type    TPoint
	Name    string
	Comment string
}

type NodeConf struct {
	Name             string
	Comment          string
	M                util.M
	Ins              []PointConf
	Outs             []PointConf
	PointToProcessor map[string]string
}

type LinkConf struct {
	OutNode  string
	OutPoint string
	InNode   string
	InPoint  string
}

type Conf struct {
	Name      string
	Comment   string
	Nodes     []NodeConf
	Links     []LinkConf
	SubGraphs []SubConf
}

type SubConf struct {
	Name      string
	Comment   string
	Nodes     []NodeConf
	Links     []LinkConf
	SubGraphs []SubConf
	In        string
	Out       string
}
