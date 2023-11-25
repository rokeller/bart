//go:build files

package files

import (
	"io"
	"os"
	"path"

	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/domain"
)

const (
	FILENAME_SETTINGS = ".settings"
	FILENAME_INDEX    = ".index.gz.encrypted"
)

type fileStorageProvider struct {
	targetRoot string
}

func NewFileStorageProvider(targetRoot string) archiving.StorageProvider {
	if err := os.MkdirAll(targetRoot, 0700); nil != err {
		glog.Exitf("Failed to create archive target directory: %v", err)
	}

	return fileStorageProvider{targetRoot: targetRoot}
}

// DeleteBackupFile implements archiving.StorageProvider.
func (p fileStorageProvider) DeleteBackupFile(entry domain.Entry) error {
	archiveRelPath := p.getArchiveRelPath(entry)
	archiveFullPath := path.Join(p.targetRoot, archiveRelPath)

	if err := os.Remove(archiveFullPath); nil != err {
		return err
	}

	// When removing parents, we don't care if that fails. After all, there could
	// still be other archived files in either one of the two parent folders
	// representing the first two bytes of the entry's RelPath hash.
	parent := path.Dir(archiveFullPath)
	if err := os.Remove(parent); nil == err {
		parent = path.Dir(parent)
		os.Remove(parent)
	}

	return nil
}

// DeleteIndex implements archiving.StorageProvider.
func (p fileStorageProvider) DeleteIndex() error {
	targetPath := path.Join(p.targetRoot, FILENAME_INDEX)
	return os.Remove(targetPath)
}

// DeleteSettings implements archiving.StorageProvider.
func (p fileStorageProvider) DeleteSettings() error {
	targetPath := path.Join(p.targetRoot, FILENAME_SETTINGS)
	return os.Remove(targetPath)
}

// NewIndexWriter implements archiving.StorageProvider.
func (p fileStorageProvider) NewIndexWriter() (io.WriteCloser, error) {
	targetPath := path.Join(p.targetRoot, FILENAME_INDEX)
	return os.Create(targetPath)
}

// NewSettingsWriter implements archiving.StorageProvider.
func (p fileStorageProvider) NewSettingsWriter() (io.WriteCloser, error) {
	targetPath := path.Join(p.targetRoot, FILENAME_SETTINGS)
	return os.Create(targetPath)
}

// ReadBackupFile implements archiving.StorageProvider.
func (p fileStorageProvider) ReadBackupFile(entry domain.Entry) (io.ReadCloser, error) {
	archiveRelPath := p.getArchiveRelPath(entry)
	archiveFullPath := path.Join(p.targetRoot, archiveRelPath)
	return os.Open(archiveFullPath)
}

// ReadIndex implements archiving.StorageProvider.
func (p fileStorageProvider) ReadIndex() (io.ReadCloser, error) {
	file, err := p.readFile(FILENAME_INDEX)

	if os.IsNotExist(err) {
		return nil, archiving.IndexNotFound
	} else if nil != err {
		return nil, err
	}

	return file, nil
}

// ReadSettings implements archiving.StorageProvider.
func (p fileStorageProvider) ReadSettings() (io.ReadCloser, error) {
	file, err := p.readFile(FILENAME_SETTINGS)

	if os.IsNotExist(err) {
		return nil, archiving.SettingsNotFound
	} else if nil != err {
		return nil, err
	}

	return file, nil
}

// WriteBackupFile implements archiving.StorageProvider.
func (p fileStorageProvider) WriteBackupFile(entry domain.Entry, file *os.File) error {
	archiveRelPath := p.getArchiveRelPath(entry)
	archiveFullPath := path.Join(p.targetRoot, archiveRelPath)
	archiveFullDir := path.Dir(archiveFullPath)

	if err := os.MkdirAll(archiveFullDir, 0700); nil != err {
		return err
	}

	targetFile, err := os.Create(archiveFullPath)
	if nil != err {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, file)
	return err
}

func (p fileStorageProvider) getArchiveRelPath(entry domain.Entry) string {
	hash := entry.Hash()
	archiveRelPath := path.Join(hash[0:2], hash[2:4], hash)

	return archiveRelPath
}

func (p fileStorageProvider) readFile(relPath string) (io.ReadCloser, error) {
	filePath := path.Join(p.targetRoot, relPath)
	return os.Open(filePath)
}
