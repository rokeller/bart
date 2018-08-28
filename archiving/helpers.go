package archiving

import "io"

type writeCloser struct {
	io.Writer
}

func (w writeCloser) Close() error {
	return nil
}
