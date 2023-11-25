package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"time"

	"github.com/golang/glog"
	"github.com/rokeller/bart/settings"
	"golang.org/x/crypto/scrypt"
)

type AesOfbContext struct {
	key []byte
}

// NewAesOfbContext creates a new crypto context.
func NewAesOfbContext(password string, s settings.Settings) AesOfbContext {
	return AesOfbContext{
		key: deriveKey(password, s),
	}
}

// Decrypt decrypts the data from the given reader and presents a reader to
// read the decrypted data from.
func (c AesOfbContext) Decrypt(r io.Reader) (io.Reader, error) {
	// At the beginning of the reader must be the unencrypted IV to use.
	iv := make([]byte, aes.BlockSize)

	if _, err := io.ReadFull(r, iv); nil != err {
		glog.Errorf("Failed to read IV from file: %v", err)
		return nil, err
	}

	// Create a new AES block cipher and init the output feedback mode stream,
	// then copy the archived file.
	blockCipher, err := aes.NewCipher(c.key)
	if nil != err {
		glog.Errorf("Failed to create AES block cipher: %v", err)
		return nil, err
	}

	stream := cipher.NewOFB(blockCipher, iv)
	decryptingReader := &cipher.StreamReader{
		S: stream,
		R: r,
	}

	return decryptingReader, nil
}

// Encrypt creates an io.WriteCloser that can be used to write data encrypted
// to the given io.Writer.
func (c AesOfbContext) Encrypt(w io.Writer) (io.WriteCloser, error) {
	// Get a new random IV and write it to the writer unencrypted.
	iv, err := getRandomIV()
	if nil != err {
		glog.Errorf("Failed to generate IV: %v", err)
		return nil, err
	}

	if _, err := w.Write(iv); nil != err {
		glog.Errorf("Failed to write IV: %v", err)
		return nil, err
	}

	// Create a new AES block cipher and init the output feedback mode stream,
	// then copy the local input file.
	blockCipher, err := aes.NewCipher(c.key)
	if nil != err {
		glog.Errorf("Failed to create AES block cipher: %v", err)
		return nil, err
	}

	stream := cipher.NewOFB(blockCipher, iv)
	encryptingWriter := &cipher.StreamWriter{
		S: stream,
		W: w,
	}

	return encryptingWriter, nil
}

func deriveKey(password string, s settings.Settings) []byte {
	startTime := time.Now()
	key, err := scrypt.Key([]byte(password), s.Salt(), 1<<18, 8, 1, 32)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	glog.Infof("Key derivation took %v", duration)

	if nil != err {
		// TODO: return error
		glog.Exitf("Failed to derive key: %v", err)
	}

	return key
}

func getRandomIV() ([]byte, error) {
	iv := make([]byte, aes.BlockSize)

	if _, err := rand.Read(iv); nil != err {
		return nil, err
	}

	return iv, nil
}
