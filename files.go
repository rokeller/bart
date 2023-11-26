//go:build files

package main

import (
	"flag"
	"os"
	"path"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/providers/files"
)

var (
	targetRoot      *string
	archiveRootPath string
)

func updateFlags(flags *flag.FlagSet) {
	targetRoot = flags.String("t", "$HOME/.backup", "The target root path for the backup.")
}

func verifyFlags() {
	archiveRootPath, _ = filepath.Abs(os.ExpandEnv(*targetRoot))
	glog.Infof("Backup archive in '%s'.", archiveRootPath)
}

func newStorageProvider(backupName string) archiving.StorageProvider {
	backupRoot := path.Join(archiveRootPath, backupName)
	provider := files.NewFileStorageProvider(backupRoot)

	return provider
}
