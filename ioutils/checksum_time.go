package ioutils

import (
	"time"
)

const DefaultTimeLayout = time.RFC822Z

func ChecksumTime(modTime time.Time) (checksum string) {
	return modTime.Format(DefaultTimeLayout)
}
