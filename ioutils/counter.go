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

package ioutils

import (
	"io"
	"sync/atomic"
)

type CountWriter struct {
	w     io.Writer
	count atomic.Int64
}

var _ io.Writer = (*CountWriter)(nil)

func (w *CountWriter) Write(b []byte) (n int, err error) {
	n, err = w.w.Write(b)
	if n > 0 {
		w.count.Add(int64(n))
	}
	return n, err
}

func (w *CountWriter) Count() (count int64) {
	return w.count.Load()
}

func NewCountWriter(w io.Writer) (c *CountWriter) {
	return &CountWriter{w: w}
}
