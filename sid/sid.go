package sid

import (
	"encoding/hex"
	"fmt"

	"github.com/15mga/kiwi/util"
	"github.com/bwmarrin/snowflake"
)

const (
	Snowflake = "snowflake"
)

var (
	_NameToFac = make(map[string]util.ToInt64)
)

func BindIdFac(name string, to util.ToInt64) {
	_NameToFac[name] = to
}

func GetIdWithName(name string) int64 {
	fac, ok := _NameToFac[name]
	if !ok {
		panic("not exist " + name)
	}
	return fac()
}

func GetStrIdWithName(name string) string {
	return hex.EncodeToString(util.Int64ToBytes(GetIdWithName(name)))
}

func SetNodeId(id int64) {
	node, err := snowflake.NewNode(id)
	if err != nil {
		panic(fmt.Sprintf("generate node failed id:%d", id))
	}
	BindIdFac(Snowflake, func() int64 {
		return node.Generate().Int64()
	})
}

func GetId() int64 {
	return GetIdWithName(Snowflake)
}

func GetStrId() string {
	return hex.EncodeToString(util.Int64ToBytes(GetIdWithName(Snowflake)))
}
