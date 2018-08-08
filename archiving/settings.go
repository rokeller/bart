package archiving

import (
	"crypto/aes"
	"crypto/rand"
	"log"
)

// Settings provides common settings used to backup and restore.
type Settings interface {
	GenerateSalt()
	GetSalt() []byte
}

type settingsBase struct {
	salt []byte

	dirty bool
}

func (s *settingsBase) GenerateSalt() {
	s.salt = make([]byte, aes.BlockSize)

	if _, err := rand.Read(s.salt); nil != err {
		log.Panicf("Failed to generate random salt: %v", err)
	}

	s.dirty = true
}
