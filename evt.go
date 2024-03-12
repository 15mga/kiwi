package kiwi

import (
	"sync"

	"github.com/15mga/kiwi/util"
)

const (
	Evt_Start           = "start"
	Evt_Stop            = "stop"
	Evt_Svc_Connected   = "svc_connected"
	Evt_Svc_Disonnected = "svc_disconnected"
)

type EvtStart struct {
	Wg *sync.WaitGroup
}

type EvtStop struct {
	Wg *sync.WaitGroup
}

type EvtRouterConnected struct {
	Svc  TSvc
	Id   int64
	Head util.M
}

type EvtRouterDisconnected struct {
	Svc TSvc
	Id  int64
}

type EvtSvcConnected struct {
	Svc  TSvc
	Id   int64
	Head util.M
}

type EvtSvcDisconnected struct {
	Svc TSvc
	Id  int64
}
