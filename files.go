//go:build files

package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/providers/files"
)

var (
	targetRoot      *string
	archiveRootPath string
)

func updateFlags() {
	targetRoot = flag.String("t", "$HOME/.backup", "The target root path for the backup.")
}

func verifyFlags() {
	archiveRootPath, _ = filepath.Abs(os.ExpandEnv(*targetRoot))
	log.Printf("Backup to '%s'.", archiveRootPath)
}

func newStorageProvider(backupName string) archiving.StorageProvider {
	backupRoot := path.Join(archiveRootPath, backupName)
	provider := files.NewFileStorageProvider(backupRoot)

	return provider
}
