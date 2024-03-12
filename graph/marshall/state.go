package marshall

import (
	"fmt"

	"github.com/15mga/kiwi/graph"
	"github.com/15mga/kiwi/util"
)

type State struct {
}

func (s *State) Marshall(grp graph.IGraph) []byte {
	var buff util.ByteBuffer
	buff.InitCap(512)
	buff.WStringNoLen("\n``` mermaid")
	buff.WStringNoLen("\nstateDiagram-v2")
	s.marshall(grp, "", &buff)
	buff.WStringNoLen("\n```")
	return buff.All()
}

func (s *State) marshall(grp graph.IGraph, prefix string, buff *util.ByteBuffer) {
	grp.IterNode(func(nd graph.INode) {
		buff.WStringNoLen(fmt.Sprintf("\n%s  %s[%s]",
			prefix, nd.Name(), nd.Comment()))
	})

	sp := prefix + "  "
	grp.IterSubGraph(func(sg graph.ISubGraph) {
		buff.WStringNoLen("\n")
		buff.WStringNoLen(fmt.Sprintf("\n%s: %s", sp, sg.Name()))
		buff.WStringNoLen(fmt.Sprintf("\n%sstate {", sp))
		s.marshall(sg, sp, buff)
		buff.WStringNoLen(fmt.Sprintf("\n%s}", sp))
		buff.WStringNoLen("\n")
	})

	grp.IterLink(func(lnk graph.ILink) {
		in := lnk.In().Node().Name()
		on := lnk.Out().Node().Name()
		typ := lnk.In().Name()
		buff.WStringNoLen(fmt.Sprintf("\n%s  %s --> |%s| %s",
			prefix, on, typ, in))
	})
}
