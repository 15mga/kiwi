package kiwi

import (
	"github.com/15mga/kiwi/sid"
	"github.com/15mga/kiwi/util"
	"time"
)

var (
	_NodeMeta = &NodeMeta{
		StartTime: time.Now().Unix(),
		Data:      util.M{},
		SvcToVer:  make(map[TSvc]*SvcConf, 8),
	}
)

func GetNodeMeta() *NodeMeta {
	return _NodeMeta
}

type NodeMeta struct {
	Ip        string
	NodeId    int64
	StartTime int64
	Data      util.M
	Mode      string
	SvcToVer  map[TSvc]*SvcConf
}

type SvcConf struct {
	Name string
	Ver  string
	Port int
}

func (n *NodeMeta) Init(id int64) {
	sid.SetNodeId(id)
	n.NodeId = sid.GetId()
	SetLogDefParams(util.M{
		"node": n.NodeId,
	})
}

func (n *NodeMeta) AddService(svc TSvc, port int, name, ver string) {
	n.SvcToVer[svc] = &SvcConf{
		Name: name,
		Ver:  ver,
		Port: port,
	}
}

func (n *NodeMeta) HasService(svc TSvc) bool {
	_, ok := n.SvcToVer[svc]
	return ok
}
