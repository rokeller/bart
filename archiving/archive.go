package archiving

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
)

// MissingFileHandler defines the contract for a handler of missing files.
type MissingFileHandler interface {
	HandleMissing(archive Archive, entry domain.Entry)
}

// Archive defines the contract for an Archive.
type Archive interface {
	Backup(ctx inspection.Context)
	Restore(entry domain.Entry)
	Delete(entry domain.Entry)

	HandleMissing(handler MissingFileHandler)

	Close()
}

type archiveBase struct {
	localRoot string

	index        Index
	missingFiles domain.BackupIndex
}

func (a *archiveBase) Close() {
	a.index.Store()
}

func (a *archiveBase) handleMissing(impl Archive, handler MissingFileHandler) {
	for relPath, meta := range a.missingFiles {
		fmt.Printf("Missing local file '%v' ... ", relPath)
		handler.HandleMissing(impl, domain.Entry{
			RelPath:       relPath,
			EntryMetadata: meta,
		})
		fmt.Println()
	}
}

func (a *archiveBase) init() *archiveBase {
	a.index.Load()

	a.missingFiles = make(domain.BackupIndex, 0)

	for key, val := range a.index.getIndex() {
		a.missingFiles[key] = val
	}

	return a
}

func (a *archiveBase) getArchivedRelFilePath(relPath string) string {
	hash := domain.GetRelPathHash(relPath)
	archiveDirPath := path.Join(hash[0:2], hash[2:4])

	return path.Join(archiveDirPath, hash)
}

func (a *archiveBase) shouldAddOrUpdate(ctx inspection.Context) bool {
	delete(a.missingFiles, ctx.RelPath())

	return a.index.shouldAddOrUpdate(ctx)

}

func (a *archiveBase) backup(ctx inspection.Context, archiveWriter io.Writer) {
	var infile *os.File
	var err error

	// Open the local file ...
	if infile, err = os.Open(ctx.AbsPath()); nil != err {
		log.Panicf("Failed to open file: %v", err)
	}

	defer infile.Close()

	// ... and copy it to the archive writer.
	if _, err = io.Copy(archiveWriter, infile); err != nil {
		log.Panicf("Failed to encrypt file: %v", err)
	}

	// Update the index to hold this new file too.
	a.index.AddOrUpdate(ctx.Entry())
}

func (a *archiveBase) restore(entry domain.Entry, archiveReader io.Reader) {
	var err error

	// Create the local directory structure and file for restoring.
	relDir := path.Dir(entry.RelPath)
	restoreDir := path.Join(a.localRoot, relDir)
	restorePath := path.Join(a.localRoot, entry.RelPath)

	if err = os.MkdirAll(restoreDir, 0700); nil != err {
		log.Panicf("Failed to create restored directory: %v", err)
	}

	var outfile *os.File

	if outfile, err = os.Create(restorePath); nil != err {
		log.Panicf("Failed to create restored file: %v", err)
	}

	defer outfile.Close()

	if _, err = io.Copy(outfile, archiveReader); err != nil {
		log.Panicf("Failed to deencrypt file: %v", err)
	}

	// Restore the timestamps to be the ones from the backup index metadata.
	ts := time.Unix(entry.Timestamp, 0)
	os.Chtimes(restorePath, ts, ts)
}

func (a *archiveBase) delete(entry domain.Entry) {
	a.index.Remove(entry.RelPath)
}
