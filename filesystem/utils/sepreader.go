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
