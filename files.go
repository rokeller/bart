//go:build files

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/rokeller/bart/archiving"
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

	if err := os.MkdirAll(archiveRootPath, 0700); nil != err {
		log.Panicf("Failed to create archive directory: %v", err)
	}
}

func newArchive(backupName, rootPath, password string) archiving.Archive {
	log.Printf("Backup '%s' as '%s' to '%s'.", rootPath, backupName, archiveRootPath)
	archive := archiving.NewFileArchive(archiveRootPath, rootPath, backupName, password)

	return archive
}
