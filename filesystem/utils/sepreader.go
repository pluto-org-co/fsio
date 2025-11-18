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

package utils

import "io"

// Handles the Read and close from two diff objects
type SeparateReadCloser struct {
	closer io.Closer
	reader io.Reader
}

func NewSeparateReadCloser(closer io.Closer, reader io.Reader) (s *SeparateReadCloser) {
	return &SeparateReadCloser{
		closer: closer,
		reader: reader,
	}
}

var (
	_ io.Closer = (*SeparateReadCloser)(nil)
	_ io.Reader = (*SeparateReadCloser)(nil)
)

func (s *SeparateReadCloser) Close() (err error) {
	return s.closer.Close()
}

func (s *SeparateReadCloser) Read(b []byte) (n int, err error) {
	return s.reader.Read(b)
}
