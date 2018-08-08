package domain

// EntryMetadata holds the metadata for a file entry in the index.
type EntryMetadata struct {
	Timestamp int64
}

// Entry holds represents an entry in the index.
type Entry struct {
	RelPath string
	EntryMetadata
}

// BackupIndex is the index of a single backup set.
type BackupIndex map[string]EntryMetadata

// Hash creates the SHA1 has for the entry's relative path.
func (e *Entry) Hash() string {
	return GetRelPathHash(e.RelPath)
}
