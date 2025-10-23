package ioutils

import (
	"io"
	"io/fs"
)

type File interface {
	fs.File
	io.ReaderAt
	Seek(offset int64, whence int) (n int64, err error)
}
