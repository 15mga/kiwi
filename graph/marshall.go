package graph

import (
	"fmt"
	"github.com/15mga/kiwi/util"
)

func LinkName(outNode, outPoint, inNode, inPoint string) string {
	return fmt.Sprintf("%s:%s->%s:%s", outNode, outPoint, inNode, inPoint)
}

type IGraphMarshaller interface {
	Marshall(g IGraph) []byte
}

type IGraphUnMarshaller interface {
	Unmarshall(bytes []byte, g IGraph) *util.Err
}

type IGraphMarshall interface {
	IGraphMarshaller
	IGraphUnMarshaller
}
