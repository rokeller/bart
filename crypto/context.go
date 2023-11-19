package crypto

import (
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/scrypt"
)

// Context defines a context for encryption.
type Context struct {
	password string
	salt     []byte

	key      []byte
	keyMutex sync.Mutex
}

// NewContext creates a new crypto context.
func NewContext(password string, salt []byte) *Context {
	return &Context{
		password: password,
		salt:     salt,
		keyMutex: sync.Mutex{},
	}
}

func (c *Context) getKey() []byte {
	if nil == c.key {
		c.keyMutex.Lock()
		defer c.keyMutex.Unlock()

		if nil == c.key {
			startTime := time.Now()
			key, err := scrypt.Key([]byte(c.password), c.salt, 1<<18, 8, 1, 32)
			endTime := time.Now()
			duration := endTime.Sub(startTime)
			log.Printf("Key derivation took %v", duration)

			if nil != err {
				log.Panicf("Failed to derive key: %v", err)
			}

			c.key = key
		}
	}

	return c.key
}
