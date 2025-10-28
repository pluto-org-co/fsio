package ioutils

import (
	"fmt"
	"time"
)

const DefaultTimeLayout = time.RFC822Z

func ChecksumTime(modTime time.Time, size int64) (checksum string) {
	return fmt.Sprintf("%s-%d", modTime.Local().Format(DefaultTimeLayout), size)
}
