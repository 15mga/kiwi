package worker

import "github.com/15mga/kiwi/util"

func Self(fn util.FnAnySlc, params ...any) {
	fn(params)
}
