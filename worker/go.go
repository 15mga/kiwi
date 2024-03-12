package worker

import (
	"github.com/15mga/kiwi/util"
	"github.com/panjf2000/ants/v2"
)

func Go(fn util.FnAnySlc, params ...any) {
	_ = ants.Submit(func() {
		fn(params)
	})
}
