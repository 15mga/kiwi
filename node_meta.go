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
		Services:  make(map[TSvc]string, 8),
	}
)

func GetNodeMeta() *NodeMeta {
	return _NodeMeta
}

type NodeMeta struct {
	Ip        string
	Port      int
	NodeId    int64
	StartTime int64
	Data      util.M
	Mode      string
	Services  map[TSvc]string
}

func (n *NodeMeta) Init(id int64) {
	sid.SetNodeId(id)
	n.NodeId = sid.GetId()
	SetLogDefParams(util.M{
		"node": n.NodeId,
	})
}

func (n *NodeMeta) AddService(svc TSvc, ver string) {
	n.Services[svc] = ver
}

func (n *NodeMeta) HasService(svc TSvc) bool {
	_, ok := n.Services[svc]
	return ok
}
