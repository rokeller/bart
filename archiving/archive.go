package archiving

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/golang/glog"
	"github.com/rokeller/bart/crypto"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/settings"
)

type Archive struct {
	localContext    LocalContext
	storageProvider StorageProvider
	settings        settings.Settings
	cryptoContext   crypto.AesOfbContext
	index           *Index
}

// NewArchive creates a new archive.
func NewArchive(password string, localContext LocalContext, storageProvider StorageProvider) Archive {
	a := Archive{
		localContext:    localContext,
		storageProvider: storageProvider,
		settings:        loadSettings(storageProvider),
	}

	a.cryptoContext = crypto.NewAesOfbContext(password, a.settings)
	a.index = newIndex(&a)
	glog.Infof("The archive index currently has %d file(s).", a.index.Count())

	return a
}

// NeedsBackup determines if the given entry needs to be backed up.
func (a Archive) NeedsBackup(entry domain.Entry) bool {
	return a.index.needsBackup(entry)
}

// Backup backs up the given entry.
func (a Archive) Backup(entry domain.Entry) error {
	absPath := path.Join(a.localContext.rootDir, entry.RelPath)

	// Open the local file ...
	src, err := os.Open(absPath)
	if nil != err {
		glog.Errorf("Failed to open local file reader: %v", err)
		return err
	}
	defer src.Close()

	w, err := a.newBackupFile(entry)
	if nil != err {
		glog.Errorf("Failed to create temporary file: %v", err)
		return err
	}
	defer w.Close()

	cw, err := a.cryptoContext.Encrypt(w)
	if nil != err {
		glog.Errorf("Failed to encrypt backup writer: %v", err)
		return err
	}
	defer cw.Close()

	// ... and copy it to the archive writer.
	if _, err = io.Copy(cw, src); err != nil {
		glog.Errorf("Failed to write to backup: %v", err)
		return err
	}

	if err := w.Upload(); nil != err {
		return err
	}

	a.index.setEntry(entry, EntryFlagsPresentInBackup|EntryFlagsPresentInLocal, true)

	return nil
}

// Restore restores the given entry.
func (a Archive) Restore(entry domain.Entry) error {
	relDir := path.Dir(entry.RelPath)
	restoreDir := path.Join(a.localContext.rootDir, relDir)
	restorePath := path.Join(a.localContext.rootDir, entry.RelPath)

	if err := os.MkdirAll(restoreDir, 0700); nil != err {
		return err
	}

	r, err := a.storageProvider.ReadBackupFile(entry)
	if nil != err {
		return err
	}
	defer r.Close()

	cr, err := a.cryptoContext.Decrypt(r)
	if nil != err {
		return err
	}

	outfile, err := os.Create(restorePath)
	if nil != err {
		return err
	}
	defer outfile.Close()

	if _, err = io.Copy(outfile, cr); err != nil {
		return err
	}

	// Restore the timestamps to be the ones from the backup index metadata.
	ts := time.Unix(entry.Timestamp, 0)
	return os.Chtimes(restorePath, ts, ts)
}

// Delete deletes the given entry from the backup.
func (a Archive) Delete(entry domain.Entry) error {
	panic("unimplemented")
}

// FindLocallyMissing finds entries that are in the backup but not available
// locally.
func (a Archive) FindLocallyMissing(fn func(entry domain.Entry)) {
	a.index.walkIndex(func(entry domain.Entry, flags EntryFlags) error {
		if flags&(EntryFlagsPresentInLocal|EntryFlagsPresentInBackup) == EntryFlagsPresentInBackup {
			fn(entry)
		}

		return nil
	})
}

// Close closes the archive.
func (a Archive) Close() error {
	return a.index.Close()
}
