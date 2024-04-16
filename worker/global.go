package worker

import "github.com/15mga/kiwi/util"

var (
	_Global *global
)

func Global() *global {
	return _Global
}

func InitGlobal() {
	if _Global != nil {
		return
	}
	_Global = NewGlobal()
	_Global.worker.Start()
}

func NewGlobal() *global {
	return &global{
		worker: NewFnWorker(1024),
	}
}

type global struct {
	worker *FnWorker
}

func (o *global) Push(fn util.FnAny, data any) {
	o.worker.Push(fn, data)
}

func (o *global) Dispose() {
	o.worker.Dispose()
}
