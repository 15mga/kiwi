package etd

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	etcd "go.etcd.io/etcd/client/v3"
)

func TestLease(t *testing.T) {
	err := Conn(etcd.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})
	assert.Nil(t, err)
	id, err := PutWithTtl("test", "95eh", 10)
	assert.Nil(t, err)
	time.Sleep(time.Second * 7)
	fmt.Println("revoke")
	_ = Revoke(id)
	time.Sleep(time.Second * 5)
}
