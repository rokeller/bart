package domain

import (
	"crypto/sha1"
	"encoding/hex"
)

// Entry holds represents an entry in the index.
type Entry struct {
	RelPath string
	EntryMetadata
}

// EntryMetadata holds the metadata for a file entry in the index.
type EntryMetadata struct {
	Timestamp int64
}

// Hash creates the SHA1 has for the entry's relative path.
func (e *Entry) Hash() string {
	return relPathHash(e.RelPath)
}

// relPathHash creates the SHA1 hash for the given relative path.
func relPathHash(relPath string) string {
	hash := sha1.Sum([]byte(relPath))
	return hex.EncodeToString(hash[0:])
}
