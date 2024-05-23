package kiwi

import "github.com/15mga/kiwi/util"

type IService interface {
	Ver() string
	Svc() TSvc
	Meta() util.M
	Start()
	AfterStart()
	Shutdown()
	Dispose()
	WatchNotice(msg util.IMsg, handler NotifyHandler)
	GetWatchCodes(svc TSvc) ([]TCode, bool)
	OnNotice(pkt IRcvNotice)
	HasNoticeWatcher(svc TSvc, code TCode) bool
}
