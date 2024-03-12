package util

type (
	poolOption[T any] struct {
		minCap int
		maxCap int
		spawn  func() T
	}
	PoolOption[T any] func(o *poolOption[T])
)

func PoolMinCap[T any](c int) PoolOption[T] {
	return func(o *poolOption[T]) {
		o.minCap = c
	}
}

func PoolMaxCap[T any](c int) PoolOption[T] {
	return func(o *poolOption[T]) {
		o.maxCap = c
	}
}

func PoolSpawn[T any](spawn func() T) PoolOption[T] {
	return func(o *poolOption[T]) {
		o.spawn = spawn
	}
}

func NewPool[T any](opts ...PoolOption[T]) *Pool[T] {
	opt := &poolOption[T]{
		minCap: 16,
		maxCap: 256,
	}
	for _, o := range opts {
		o(opt)
	}
	return &Pool[T]{
		opt:    opt,
		cap:    opt.minCap,
		values: make([]T, opt.minCap),
	}
}

type Pool[T any] struct {
	opt    *poolOption[T]
	values []T
	idx    int
	cap    int
}

func (p *Pool[T]) Shift() (v T) {
	if p.idx == 0 {
		v = p.opt.spawn()
		return
	}
	p.idx--
	v = p.values[p.idx]
	return
}

func (p *Pool[T]) Push(value T) {
	if p.idx == p.cap {
		if p.cap == p.opt.maxCap {
			return
		}
		p.resetCap(p.cap << 1)
	}
	p.values[p.idx] = value
	p.idx++
}

func (p *Pool[T]) resetCap(c int) {
	p.cap = c
	values := make([]T, p.cap)
	copy(values, p.values[:p.idx])
	p.values = values
}
