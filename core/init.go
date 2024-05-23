package core

import (
	"fmt"
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
	fmt.Println("email:", "15m@15m.games")
	fmt.Println("site:", "https://15m.games/category/Kiwi")
	fmt.Println("path:", util.WorkDir()+"/"+util.ExeName())
	fmt.Println("time:", time.Now().Format(time.DateTime))
}
