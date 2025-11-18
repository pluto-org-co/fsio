// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
