package archiving

import (
	"log"

	"github.com/rokeller/bart/settings"
)

func loadSettings(p StorageProvider) settings.Settings {
	r, err := p.ReadSettings()
	if nil != err {
		if _, ok := err.(SettingsNotFound); ok {
			log.Println("Settings not found, creating new settings.")
			settings := settings.NewSettings()

			err = storeSettings(p, settings)
			if nil != err {
				log.Fatalf("Settings could not be written to backup destination: %v", err)
			}

			return settings
		}

		log.Fatalf("Failed to load archive settings: %v", err)
	}
	defer r.Close()

	settings, err := settings.NewSettingsFromReader(r)
	if nil != err {
		log.Fatalf("Failed to read settings: %v", err)
	}

	return settings
}

func storeSettings(p StorageProvider, s settings.Settings) error {
	w, err := p.NewSettingsWriter()
	if nil != err {
		return err
	}
	defer w.Close()

	return s.Write(w)
}
