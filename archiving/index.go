package archiving

import (
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
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
