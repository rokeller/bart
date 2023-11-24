package archiving

import (
	"log"
	"os"

	"github.com/rokeller/bart/domain"
)

type backupFile struct {
	a Archive
	e domain.Entry
	f *os.File
}

func (a Archive) newBackupFile(entry domain.Entry) (*backupFile, error) {
	tempFile, err := os.CreateTemp(os.TempDir(), "bart-*")
	if nil != err {
		log.Printf("Failed to create temp file: %v", err)
		return nil, err
	}

	f := backupFile{
		a: a,
		e: entry,
		f: tempFile,
	}

	return &f, nil
}

// Write implements io.WriteCloser.
func (f backupFile) Write(p []byte) (n int, err error) {
	return f.f.Write(p)
}

// Close implements io.WriteCloser.
func (f backupFile) Close() error {
	defer os.Remove(f.f.Name())

	return f.f.Close()
}

func (f backupFile) Upload() error {
	// The file has just been written to. We need to seek to the beginning
	// before we can upload bytes from it.
	_, err := f.f.Seek(0, 0)
	if nil != err {
		return err
	}

	return f.a.storageProvider.WriteBackupFile(f.e, f.f)
}
