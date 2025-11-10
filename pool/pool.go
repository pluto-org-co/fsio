package pool

import "sync"

type Pool[T any] struct {
	pool sync.Pool
}

func (p *Pool[T]) Get() (v *T) {
	return p.pool.Get().(*T)
}

func (p *Pool[T]) Put(v *T) {
	p.pool.Put(v)
}

func New[T any]() (pool *Pool[T]) {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return new(T) },
		},
	}
}

func NewWithFunc[T any](f func() (v *T)) (pool *Pool[T]) {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return f() },
		},
	}
}
