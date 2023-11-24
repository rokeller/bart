package archiving

import (
	"io"
	"os"

	"github.com/rokeller/bart/domain"
)

// SettingsNotFound defines the error that is raised when an existing archive's
// settings cannot be located.
type SettingsNotFound struct{}

func (SettingsNotFound) Error() string {
	return "the settings were not found in the backup destination"
}

// IndexNotFound defines the error that is raised when an existing archive index
// cannot be located.
type IndexNotFound struct{}

func (IndexNotFound) Error() string {
	return "the index was not found in the backup destination"
}

type StorageProvider interface {
	// When the backup destination does not have settings yet, the error must
	// be archiving.SettingsNotFound{}.
	ReadSettings() (io.ReadCloser, error)
	// When the backup destination does not have an index yet, the error must
	// be archiving.IndexNotFound{}.
	ReadIndex() (io.ReadCloser, error)
	ReadBackupFile(entry domain.Entry) (io.ReadCloser, error)

	NewSettingsWriter() (io.WriteCloser, error)
	NewIndexWriter() (io.WriteCloser, error)
	WriteBackupFile(entry domain.Entry, file *os.File) error

	DeleteSettings() error
	DeleteIndex() error
	DeleteBackupFile(entry domain.Entry) error
}

type LocalContext struct {
	rootDir string
}

func NewLocalContext(rootDir string) LocalContext {
	return LocalContext{
		rootDir: rootDir,
	}
}
