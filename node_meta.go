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
	}
)

func GetNodeMeta() *NodeMeta {
	return _NodeMeta
}

type NodeMeta struct {
	Id        int64
	Ip        string
	Port      int
	NodeId    int64
	StartTime int64
	Mode      string
	Data      util.M
}

func (n *NodeMeta) Init(id int64) {
	if id < 0 {
		Fatal(util.NewErr(util.EcParamsErr, util.M{
			"id": id,
		}))
	}
	n.Id = id
	sid.SetNodeId(id)
	n.NodeId = sid.GetId()
	SetLogDefParams(util.M{
		"node": n.NodeId,
	})
}
