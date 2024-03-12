package core

const (
	HdHeartbeat uint8 = iota + 1
	HdPush
	HdRequest
	HdOk
	HdFail
	HdWatch
	HdNotify
)

var (
	Heartbeat = []byte{HdHeartbeat}
)
