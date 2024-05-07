package core

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
	"runtime"
	"time"
)

const (
	_Logo = ` 
 __  __ _______ ________ _______ 
|  |/  |_     _|  |  |  |_     _|
|     < _|   |_|  |  |  |_|   |_ 
|__|\__|_______|________|_______|
`
)

func init() {
	fmt.Println(_Logo)
	fmt.Println("ver:", runtime.Version())
	fmt.Println("auth:", "95eh")
	fmt.Println("email:", "eh95@qq.com")
	fmt.Println("site:", "https://15m.games/category/Kiwi")
	fmt.Println("path:", util.WorkDir()+"/"+util.ExeName())
	fmt.Println("time:", time.Now().Format(time.DateTime))
}

var (
	_Services = ds.NewKSet[kiwi.TSvc, kiwi.IService](8, func(service kiwi.IService) kiwi.TSvc {
		return service.Svc()
	})
)

func RegisterSvc(services ...kiwi.IService) {
	for _, service := range services {
		_ = _Services.Add(service)
	}
}

func StartAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.Start()
	})
}

func AfterStartAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.AfterStart()
	})
}

func ShutdownAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.Shutdown()
	})
}

func GetService(svc kiwi.TSvc) (kiwi.IService, bool) {
	return _Services.Get(svc)
}
