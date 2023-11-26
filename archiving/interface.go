package archiving

import (
	"errors"
	"io"
	"os"

	"github.com/rokeller/bart/domain"
)

// SettingsNotFound defines the error that is raised when an existing archive's
// settings cannot be located.
var SettingsNotFound = errors.New("the settings were not found in the backup destination")

// IndexNotFound defines the error that is raised when an existing archive index
// cannot be located.
var IndexNotFound = errors.New("the index was not found in the backup destination")

// IndexDecryptionFailed defines the error that is raised when the decryption of
// an archive's index failed, most likely due to a wrong crypto key.
var IndexDecryptionFailed = errors.New("decryption of the archive index failed")

// BackupFileNotFound defines the error that is raised when a file is not found
// in the backup archive.
var BackupFileNotFound = errors.New("the file was not found in the backup")

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
