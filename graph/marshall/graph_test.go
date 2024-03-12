package marshall

import (
	"github.com/15mga/kiwi/graph"
	"github.com/15mga/kiwi/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

var str = `
graph TD
  signIn[登录]
  signUp[注册]
	characterCreate[创建角色]
	characterGet[获取角色]
	characterEquipChange[更换装备]
	roomList[获取房间列表]
	roomEnter[进入房间]
	sceneCharacterUpdate[角色位置更新]
	roomExit[退出房间]
  over[结束]
  signUp --> |fail->error->fail| over
  signUp --> |success->empty->create| characterCreate
	signIn --> |success->empty->get| characterGet
  signIn --> |fail->account->signUp| signUp
	characterCreate --> |success->empty->get| characterGet
	characterCreate --> |fail->error->fail| over
	characterGet --> |success->empty->change| characterEquipChange
	characterGet --> |success->empty->end| over
	characterEquipChange --> |success->empty->get| roomList
	characterEquipChange --> |fail->empty->get| roomList
	roomList --> |success->int64->enter| roomEnter
	roomList --> |fail->error->fail| over
	roomEnter --> |success->empty->update| sceneCharacterUpdate
	sceneCharacterUpdate --> |success->empty->update| roomExit
	roomExit --> |success->empty->get| roomList
	roomExit --> |over->empty->end| over`

func TestGraph_Unmarshall(t *testing.T) {
	g := graph.NewGraph("test")
	ug := &Graph{}
	err := ug.Unmarshall(util.StrToBytes(str), g)
	assert.Nil(t, err)
}
