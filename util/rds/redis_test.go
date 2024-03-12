package rds

import (
	"fmt"
	"testing"

	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/gomodule/redigo/redis"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

func init() {
	fac := func() (redis.Conn, error) {
		return redis.Dial("tcp", "127.0.0.1:6379",
			redis.DialDatabase(0))
	}
	InitRedis(
		ConnFac(fac),
	)
}

func TestJsonGet(t *testing.T) {
	conn, err := GetConn()
	assert.Nil(t, err)
	defer conn.Close()

	si := kiwi.NodeMeta{
		Svc:   1,
		Ip:    "127.0.0.1",
		Port:  7300,
		SvcId: 10000,
	}

	key := "service:register:1:gate"
	err = JsonSet(conn, key, util.M{
		"1001": si,
	})
	assert.Nil(t, err)

	var m map[string]*kiwi.NodeMeta
	err = JsonGet[map[string]*kiwi.NodeMeta](conn, key, &m)
	assert.Nil(t, err)
	assert.Equal(t, map[string]*kiwi.NodeMeta{
		"1001": &si,
	}, m)

	key = "test:2"
	err = JsonSet(conn, key, util.M{
		"name": "95eh",
	})
	assert.Nil(t, err)
	var m1 map[string]string
	err = JsonGet[map[string]string](conn, key, &m1)
	assert.Equal(t, m1, map[string]string{
		"name": "95eh",
	})
	assert.Nil(t, err)

	var names string
	err = JsonGet[string](conn, key, &names, "name")
	assert.Equal(t, names, "95eh")
	assert.Nil(t, err)
}

func TestScan(t *testing.T) {
	conn, err := GetConn()
	assert.Nil(t, err)
	defer conn.Close()

	em := map[string]util.M{
		"service:alias:100": {
			"scene": "world0",
		},
		"service:alias:101": {
			"scene": "novice0",
		},
	}
	for key, m := range em {
		slc := []any{key}
		m.ToSlice(&slc)
		e := conn.Send(HSET, slc...)
		assert.Nil(t, e)
	}
	e := conn.Flush()
	assert.Nil(t, e)
	err = Scan(conn, "service:alias:*", 100, func(keys []string) {
		for i, key := range keys {
			fmt.Println(i, key)
			e := conn.Send(HGETALL, key)
			assert.Nil(t, e)
			keys[i] = key
		}
		e := conn.Flush()
		assert.Nil(t, e)
		for _, key := range keys {
			m, e := redis.StringMap(conn.Receive())
			assert.Nil(t, e)
			var um util.M
			e = mapstructure.Decode(m, &um)
			assert.Nil(t, e)
			assert.EqualValues(t, um, em[key])
		}
	})
	assert.Nil(t, err)
}

func TestTmp(t *testing.T) {
	conn, err := GetConn()
	assert.Nil(t, err)
	defer conn.Close()

	kvs := []any{"test"}
	util.M{
		"name": "95eh",
		"city": "wuhan",
	}.ToSlice(&kvs)
	_, e := conn.Do(HSET, kvs...)
	fmt.Println(e)
}
