package domain

import (
	"crypto/sha1"
	"encoding/hex"
)

// GetRelPathHash creates the SHA1 hash for the given relative path.
func GetRelPathHash(relPath string) string {
	hash := sha1.Sum([]byte(relPath))
	return hex.EncodeToString(hash[0:])
}
