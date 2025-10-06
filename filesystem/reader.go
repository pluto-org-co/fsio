package filesystem

import "io"

type separateReadCloser struct {
	closer io.Closer
	reader io.Reader
}

var (
	_ io.Closer = (*separateReadCloser)(nil)
	_ io.Reader = (*separateReadCloser)(nil)
)

func (s *separateReadCloser) Close() (err error) {
	return s.closer.Close()
}

func (s *separateReadCloser) Read(b []byte) (n int, err error) {
	return s.reader.Read(b)
}
