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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type SelfdestructionFile struct {
	*os.File
}

func (f *SelfdestructionFile) Seek(offset int64, whence int) (n int64, err error) {
	return f.File.Seek(offset, whence)
}

func (f *SelfdestructionFile) Close() (err error) {
	f.File.Close()
	os.Remove(f.File.Name())
	return nil
}

// Save a temporary file in order to prevent connection timeout or similar error from unknown readers.
// This functions creates the temporary file and automatically cleans on close.
// If reader is actually a *os.File, the function will return it without managing its deletion
func ReaderToTempFile(ctx context.Context, src io.Reader) (file File, err error) {
	switch tr := src.(type) {
	case *os.File:
		return tr, nil
	default:
		temp, err := os.CreateTemp("", "*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file: %w", err)
		}
		defer func() {
			if err != nil {
				temp.Close()
				os.Remove(temp.Name())
			}
		}()
		go func() {
			<-ctx.Done()
			temp.Close()
			os.Remove(temp.Name())
		}()

		writer := bufio.NewWriter(temp)
		var reader io.Reader
		switch src.(type) {
		case *bufio.Reader:
			reader = src
		default:
			reader = bufio.NewReader(src)
		}

		_, err = CopyContext(ctx, writer, reader, DefaultBufferSize)
		if err != nil {
			return nil, fmt.Errorf("failed to copy contents: %w", err)
		}

		err = writer.Flush()
		if err != nil {
			return nil, fmt.Errorf("failed to flush writer: %w", err)
		}

		_, err = temp.Seek(0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to the begining of the file: %w", err)
		}

		return &SelfdestructionFile{File: temp}, nil
	}
}
