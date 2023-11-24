package settings

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"

	"github.com/rokeller/bart/domain"
	"google.golang.org/protobuf/proto"
)

type Settings struct {
	salt []byte
}

// NewSettings generates new settings with a new salt etc.
func NewSettings() Settings {
	salt := make([]byte, aes.BlockSize)

	if _, err := rand.Read(salt); nil != err {
		log.Fatalf("Failed to generate random salt: %v", err)
	}

	return Settings{
		salt: salt,
	}
}

func NewSettingsFromReader(r io.ReadCloser) (Settings, error) {
	settingsSize := make([]byte, 4)
	_, err := io.ReadFull(r, settingsSize)

	if nil != err {
		return Settings{}, err
	}

	dataSize := binary.LittleEndian.Uint32(settingsSize)
	data := make([]byte, dataSize)
	_, err = io.ReadFull(r, data)

	if nil != err {
		return Settings{}, err
	}

	settings := &domain.Settings{}

	if err := proto.Unmarshal(data, settings); nil != err {
		log.Panicf("Failed to unmarshal settings: %v", err)
	}

	return Settings{
		salt: settings.Salt,
	}, nil
}

func (s Settings) Salt() []byte {
	return s.salt
}

func (s Settings) Write(w io.Writer) error {
	settings := &domain.Settings{
		Salt: s.salt,
	}

	data, err := proto.Marshal(settings)
	if nil != err {
		log.Printf("Failed to marshal settings: %v", err)
		return err
	}

	settingsSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(settingsSize, uint32(len(data)))

	if _, err := w.Write(settingsSize); nil != err {
		log.Printf("Failed to write settings size: %v", err)
		return err
	}

	if _, err := w.Write(data); nil != err {
		log.Printf("Failed to write settings data: %v", err)
		return err
	}

	return nil
}
