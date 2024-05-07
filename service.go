package kiwi

type IService interface {
	Svc() TSvc
	Start()
	AfterStart()
	Shutdown()
	Dispose()
}
