package ioutils

import "io/fs"

type File interface {
	fs.File
	Seek(offset int64, whence int) (n int64, err error)
}
