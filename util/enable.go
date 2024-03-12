package util

import (
	"sync"
)

type Enable struct {
	Mtx      sync.RWMutex
	disabled bool
}

func NewEnable() *Enable {
	return &Enable{
		disabled: true,
	}
}

func (e *Enable) Disabled() bool {
	return e.disabled
}

func (e *Enable) RAction(fn FnAnySlc, params ...any) *Err {
	if fn == nil {
		return nil
	}
	e.Mtx.RLock()
	if e.disabled {
		e.Mtx.RUnlock()
		return NewErr(EcClosed, nil)
	}
	fn(params)
	e.Mtx.RUnlock()
	return nil
}

func (e *Enable) WAction(fn FnAnySlc, params ...any) *Err {
	if fn == nil {
		return nil
	}
	e.Mtx.Lock()
	if e.disabled {
		e.Mtx.Unlock()
		return NewErr(EcClosed, nil)
	}
	fn(params)
	e.Mtx.Unlock()
	return nil
}

func (e *Enable) WAction2(fn Fn) *Err {
	if fn == nil {
		return nil
	}
	e.Mtx.Lock()
	if e.disabled {
		e.Mtx.Unlock()
		return NewErr(EcClosed, nil)
	}
	fn()
	e.Mtx.Unlock()
	return nil
}

func (e *Enable) Disable(fn FnAnySlc, params ...any) bool {
	e.Mtx.Lock()
	if e.disabled {
		e.Mtx.Unlock()
		return false
	}
	e.disabled = true
	if fn != nil {
		fn(params)
	}
	e.Mtx.Unlock()
	return true
}

func (e *Enable) Enable(fn FnAnySlc, params ...any) bool {
	e.Mtx.Lock()
	if !e.disabled {
		e.Mtx.Unlock()
		return false
	}
	e.disabled = false
	if fn != nil {
		fn(params)
	}
	e.Mtx.Unlock()
	return true
}

func (e *Enable) IfDisable(fn Fn) (ok bool) {
	if fn == nil {
		ok = true
		return
	}
	e.Mtx.RLock()
	if e.disabled {
		ok = true
		if fn != nil {
			fn()
		}
	}
	e.Mtx.RUnlock()
	return
}

func (e *Enable) IfEnable(fn Fn) (ok bool) {
	if fn == nil {
		ok = true
		return
	}
	e.Mtx.RLock()
	if !e.disabled {
		ok = true
		if fn != nil {
			fn()
		}
	}
	e.Mtx.RUnlock()
	return
}
