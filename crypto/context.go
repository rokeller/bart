package crypto

import (
	"log"
	"time"

	"golang.org/x/crypto/scrypt"
)

// Context defines a context for encryption.
type Context struct {
	password string
	salt     []byte

	key []byte
}

// NewContext creates a new crypto context.
func NewContext(password string, salt []byte) *Context {
	return &Context{
		password: password,
		salt:     salt,
	}
}

func (a *Context) getKey() []byte {
	if nil == a.key {
		startTime := time.Now()
		key, err := scrypt.Key([]byte(a.password), a.salt, 1<<18, 8, 1, 32)
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		log.Printf("Key derivation took %v", duration)

		if nil != err {
			log.Panicf("Failed to derive key: %v", err)
		}

		a.key = key
	}

	return a.key
}
