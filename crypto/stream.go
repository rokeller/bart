package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
)

// NewCryptoReader creates a new io.Reader around the given io.Reader.
func NewCryptoReader(ctx *Context, r io.Reader) io.Reader {
	// At the beginning of the reader must be the unencrypted IV to use.
	iv := make([]byte, aes.BlockSize)

	if _, err := io.ReadFull(r, iv); nil != err {
		log.Panicf("Failed to read IV from file: %v", err)
	}

	// Create a new AES block cipher and init the output feedback mode stream, then copy the archived file.
	blockCipher, err := aes.NewCipher(ctx.getKey())

	if nil != err {
		log.Panicf("Failed to create AES block cipher: %v", err)
	}

	stream := cipher.NewOFB(blockCipher, iv)

	decryptingReader := &cipher.StreamReader{
		S: stream,
		R: r,
	}

	return decryptingReader
}

// NewCryptoWriter creates a new io.Writer around the given io.Writer.
func NewCryptoWriter(ctx *Context, w io.Writer) io.Writer {
	// Get a new random IV and write it to the writer unencrypted.
	iv := getRandomIV()

	if _, err := w.Write(iv); nil != err {
		log.Panicf("Failed to write IV: %v", err)
	}

	// Create a new AES block cipher and init the output feedback mode stream, then copy the local input file.
	blockCipher, err := aes.NewCipher(ctx.getKey())

	if nil != err {
		log.Panicf("Failed to create AES block cipher: %v", err)
	}

	stream := cipher.NewOFB(blockCipher, iv)

	encryptingWriter := &cipher.StreamWriter{
		S: stream,
		W: w,
	}

	return encryptingWriter
}

func getRandomIV() []byte {
	iv := make([]byte, aes.BlockSize)

	if _, err := rand.Read(iv); nil != err {
		log.Panicf("Failed to generate IV: %v", err)
	}

	return iv
}
