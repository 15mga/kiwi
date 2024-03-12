package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
)

func InitNodeTest() {
	kiwi.SetNode(&nodeTest{
		nodeLocal: nodeLocal{newNodeBase()},
	})
}

type nodeTest struct {
	nodeLocal
}

func (n *nodeTest) Push(pus kiwi.ISndPush) {
	kiwi.Debug("push", util.M{
		"pid":  pus.Pid(),
		"tid":  pus.Tid(),
		"svc":  pus.Svc(),
		"code": pus.Code(),
		"head": pus.Head(),
		"msg":  pus.Msg(),
	})
}

func (n *nodeTest) PushNode(nodeId int64, pus kiwi.ISndPush) {
	//kiwi.Debug("push node", util.M{
	//	"node id": nodeId,
	//	"pid":     pus.Pid(),
	//	"tid":     pus.Tid(),
	//	"svc":     pus.Svc(),
	//	"code":    pus.Code(),
	//	"head":    pus.Head(),
	//	"msg":     pus.Msg(),
	//})
}
