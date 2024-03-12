package dynamo

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/log"
	"github.com/15mga/kiwi/util"
	"testing"
)

func TestConn(t *testing.T) {
	kiwi.SetLogger(
		log.NewStd(
			log.StdLogLvl(kiwi.LvlToMask(kiwi.TestLevels...)),
		),
	)
	err := ConnLocal("http://192.168.3.35:8000")
	if err != nil {
		kiwi.Error(err)
		return
	}
	exist, err := IsTableExist("account")
	if err != nil {
		kiwi.Error(err)
		return
	}
	kiwi.Debug("account table", util.M{
		"exist": exist,
	})
}
