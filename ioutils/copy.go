package ioutils

import (
	"context"
	"errors"
	"fmt"
	"io"
)

func CopyContext(ctx context.Context, dst io.Writer, src io.Reader, size int64) (n int64, err error) {
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return n, fmt.Errorf("context error during copy: %w", err)
			}
			return n, nil
		default:
			chunkCopy, err := io.CopyN(dst, src, size)
			n += chunkCopy
			if err != nil {
				if errors.Is(err, io.EOF) {
					return n, nil
				}
				return n, fmt.Errorf("failed to copy chunk: %w: could write at least %d", err, n)
			}
		}
	}
}
