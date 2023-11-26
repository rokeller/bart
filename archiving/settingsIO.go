package archiving

import (
	"github.com/golang/glog"
	"github.com/rokeller/bart/settings"
)

func loadSettings(p StorageProvider) settings.Settings {
	r, err := p.ReadSettings()
	if nil != err {
		if err == SettingsNotFound {
			glog.Info("Settings not found, creating new settings.")
			settings := settings.NewSettings()

			err = storeSettings(p, settings)
			if nil != err {
				glog.Exitf("Settings could not be written to backup destination: %v", err)
			}

			return settings
		}

		glog.Exitf("Failed to load archive settings: %v", err)
	}
	defer r.Close()

	settings, err := settings.NewSettingsFromReader(r)
	if nil != err {
		glog.Exitf("Failed to read settings: %v", err)
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
