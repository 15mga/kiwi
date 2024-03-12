package graph

import (
	"fmt"
	"github.com/15mga/kiwi"
	"strings"
	"testing"
	"time"

	"github.com/15mga/kiwi/util"
	"github.com/15mga/kiwi/worker"
	"github.com/stretchr/testify/assert"
)

const (
	PTimout = "timeout"
)

func NewFsmPlugin() *fsmPlugin {
	return &fsmPlugin{
		pathToTimeout: make(map[string]time.Duration),
		manualDisable: make(map[string]struct{}),
		autoEnable:    make(map[string]INode),
		worker:        worker.NewFnWorker(),
	}
}

type fsmPlugin struct {
	worker        *worker.FnWorker
	pathToTimeout map[string]time.Duration //激活节点超时关闭节点
	manualDisable map[string]struct{}      //手动关闭,节点输出后不自动关闭
	autoEnable    map[string]INode
}

func (s *fsmPlugin) SetNodeTimeout(dur time.Duration, path ...string) {
	s.pathToTimeout[strings.Join(path, ".")] = dur
}

func (s *fsmPlugin) SetManualDisable(path ...string) {
	s.manualDisable[strings.Join(path, ".")] = struct{}{}
}

func (s *fsmPlugin) SetAutoEnable(path ...string) {
	s.autoEnable[strings.Join(path, ".")] = nil
}

func (s *fsmPlugin) OnStart(g IGraph) {
	s.worker.Start()
}

func (s *fsmPlugin) OnAddIn(in IIn) {
	in.AddFilter(func(msg IMsg) *util.Err {
		inNode := in.Node()
		_, ok := s.manualDisable[inNode.Path()]
		if !ok {
			err := inNode.SetEnable(true)
			if err != nil {
				kiwi.Error(err)
			}
		}
		return nil
	})
}

func (s *fsmPlugin) OnAddOut(out IOut) {
	out.AddFilter(func(msg IMsg) *util.Err {
		outNode := out.Node()
		_, ok := s.manualDisable[outNode.Path()]
		if !ok {
			err := outNode.SetEnable(false)
			if err != nil {
				kiwi.Error(err)
			}
		}
		return nil
	})
}

func (s *fsmPlugin) OnAddNode(g IGraph, nd INode) {
	nd.SetData(util.M{})

	dur, ok := s.pathToTimeout[nd.Path()]
	if ok {
		_ = nd.AddOut(TpNone, PTimout)
		nd.AddAfterEnable(func(enable bool) {
			if enable {
				nd.Data().Set(PTimout, time.AfterFunc(dur, func() {
					s.worker.Push(func(params []any) {
						nd := params[0].(INode)
						err := nd.Out(PTimout, nil)
						if err != nil {
							kiwi.Error(err)
						}
						nd.Data().Del(PTimout)
					}, nd)
				}))
			} else {
				timer, exit := util.MPop[time.Timer](nd.Data(), PTimout)
				if exit {
					timer.Stop()
				}
			}
		})
	}

	_, ok = s.autoEnable[nd.Path()]
	_ = nd.SetEnable(ok)
}

func (s *fsmPlugin) OnAddSubGraph(g IGraph, sg ISubGraph) {

}

func (s *fsmPlugin) OnAddLink(g IGraph, lnk ILink) {

}

func newFsmGraph() (IGraph, *fsmPlugin) {
	fsm := NewFsmPlugin()
	fsm.SetNodeTimeout(time.Second, "init")
	fsm.SetAutoEnable("init")
	g := NewGraph("test", Plugin(fsm))
	return g, fsm
}

func TestFsmPlugin_SetNodeTimeout(t *testing.T) {
	g, _ := newFsmGraph()
	node, err := g.AddNode("init")
	assert.Nil(t, err)
	assert.NotNil(t, node)

	op, err := node.GetOut(PTimout)
	assert.Nil(t, err)
	assert.NotNil(t, op)

	ch := make(chan struct{})
	timer := time.NewTimer(time.Millisecond * 1500)
	startTime := time.Now()
	op.AddFilter(func(msg IMsg) *util.Err {
		bytes, err := msg.ToJson()
		assert.Nil(t, err)
		fmt.Println(util.BytesToStr(bytes), time.Since(startTime).Milliseconds())
		ch <- struct{}{}
		return nil
	})
	_ = g.Start()
	select {
	case <-timer.C:
		t.Error("timeout")
	case <-ch:
		timer.Stop()
	}
}
