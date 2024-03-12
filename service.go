package kiwi

type IService interface {
	Svc() TSvc
	Start()
	Shutdown()
	Dispose()
}
