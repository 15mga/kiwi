package util

import "context"

var (
	_Ctx, _Cancel = context.WithCancel(context.Background())
)

func Ctx() context.Context {
	return _Ctx
}

func Cancel() {
	_Cancel()
}
