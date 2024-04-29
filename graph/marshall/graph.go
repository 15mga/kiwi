package marshall

import (
	"bufio"
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/graph"
	"github.com/15mga/kiwi/util"
	"regexp"
	"strings"
)

type Graph struct {
}

func (g *Graph) Marshall(grp graph.IGraph) []byte {
	var buff util.ByteBuffer
	buff.InitCap(512)
	buff.WStringNoLen("\n``` mermaid")
	buff.WStringNoLen("\ngraph TD")
	g.marshall(grp, "", &buff)
	buff.WStringNoLen("\n```")
	return buff.All()
}

func (g *Graph) marshall(grp graph.IGraph, prefix string, buff *util.ByteBuffer) {
	grp.IterNode(func(nd graph.INode) {
		buff.WStringNoLen(fmt.Sprintf("\n%s  %s[%s]",
			prefix, nd.Name(), nd.Comment()))
	})

	sp := prefix + "  "
	grp.IterSubGraph(func(sg graph.ISubGraph) {
		buff.WStringNoLen("\n")
		buff.WStringNoLen(fmt.Sprintf("\n%ssubgraph %s", sp, sg.Name()))
		g.marshall(sg, sp, buff)
		buff.WStringNoLen(fmt.Sprintf("\n%send", sp))
		buff.WStringNoLen("\n")
	})

	grp.IterLink(func(lnk graph.ILink) {
		in := lnk.In().Node().Name()
		ip := lnk.In().Name()
		typ := lnk.In().Type()
		on := lnk.Out().Node().Name()
		op := lnk.Out().Name()
		buff.WStringNoLen(fmt.Sprintf("\n%s  %s --> |%s->%s->%s| %s",
			prefix, on, op, typ, ip, in))
	})
}

func (g *Graph) Unmarshall(bytes []byte, grp graph.IGraph) *util.Err {
	reader := bufio.NewReader(strings.NewReader(util.BytesToStr(bytes)))
	ns := regexp.MustCompile(`(\[)|(])|(\()|(\))`)
	ls := regexp.MustCompile(`(-->)|(\|)`)
	ls2 := regexp.MustCompile(`->`)
	g.unmarshal(grp, reader, ns, ls, ls2)
	return nil
}

func (g *Graph) unmarshal(grp graph.IGraph, reader *bufio.Reader, ns, ls, ls2 *regexp.Regexp) {
	linkLines := make([]string, 0, 32)
	for {
		lb, _, e := reader.ReadLine()
		if e != nil {
			break
		}
		line := strings.TrimSpace(string(lb))
		if line == "" ||
			strings.HasPrefix(line, "graph TD") ||
			strings.HasPrefix(line, "subgraph") ||
			strings.HasPrefix(line, "end") ||
			strings.HasPrefix(line, "%%") ||
			strings.HasPrefix(line, "```") {
			continue
		}
		//连接
		if strings.IndexAny(line, "-->") > -1 {
			linkLines = append(linkLines, line)
			continue
		}
		//节点
		slc := ns.Split(line, -1)
		slc = filterEmptyItem(slc)
		nd := grp.GetNode(slc[0])
		if len(slc) == 2 {
			nd.SetComment(slc[1])
		}
	}
	for _, line := range linkLines {
		slc := ls.Split(line, -1)
		slc = filterEmptyItem(slc)
		on := slc[0]
		in := slc[2]
		slc2 := ls2.Split(slc[1], -1)
		slc2 = filterEmptyItem(slc2)
		var ip string
		op := slc2[0]
		t := graph.TPoint("nil")
		switch len(slc2) {
		case 2:
			ip = slc2[1]
		case 3:
			ip = slc2[2]
			t = graph.TPoint(slc2[1])
		default:
			kiwi.Error2(util.EcParseErr, util.M{
				"link": slc[1],
			})
		}
		outNode := grp.GetNode(on)
		inNode := grp.GetNode(in)
		if !outNode.HasOut(op) {
			_ = outNode.AddOut(t, op)
		}
		if !inNode.HasIn(ip) {
			_ = inNode.AddIn(t, ip)
		}
		_, err := grp.Link(on, op, in, ip)
		if err != nil {
			kiwi.Error(err)
		}
	}
}

func filterEmptyItem(slc []string) []string {
	ns := make([]string, 0, len(slc))
	for _, item := range slc {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		ns = append(ns, s)
	}
	return ns
}
