package archiving

import (
	"backup/domain"
	"backup/inspection"
)

// Index manages the index of an archive
type Index interface {
	Load()
	Store()

	AddOrUpdate(entry domain.Entry)
	Remove(relPath string)

	NumEntriesRead() int
	NumEntriesWritten() int

	shouldAddOrUpdate(ctx inspection.Context) bool
	getIndex() domain.BackupIndex
}
