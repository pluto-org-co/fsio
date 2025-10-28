package ioutils

import (
	"fmt"
	"time"
)

func ChecksumTime(modTime time.Time, size int64) (checksum string) {
	return fmt.Sprintf("%d-%d", modTime.Unix(), size)
}
