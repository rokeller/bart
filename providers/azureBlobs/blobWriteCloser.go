//go:build azure || azurite

package azureBlobs

import (
	"io"
	"sync"
)

type blobWriteCloser struct {
	w  io.WriteCloser
	wg *sync.WaitGroup
}

// Close implements io.WriteCloser.
func (w blobWriteCloser) Close() error {
	err := w.w.Close()
	w.wg.Wait()
	return err
}

// Write implements io.WriteCloser.
func (w blobWriteCloser) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}
