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
