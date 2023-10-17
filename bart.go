package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/inspection"

	"github.com/howeyc/gopass"
)

func main() {
	name := flag.String("name", "backup", "The name of the backup archive.")
	root := flag.String("path", ".", "The path to the directory to backup and/or restore.")
	missingBehavior := flag.String("m", "noop", "A behavior for files missing locally: 'noop' to do nothing, 'restore' to restore them from the backup, 'delete' to delete them in the backup archive.")
	updateFlags()
	flag.Parse()

	backupName := strings.TrimSpace(*name)

	if "" == backupName {
		log.Fatalf("The backup name must not be empty.")
	} else {
		verifyFlags()
	}

	password := readPassword()
	rootPath, _ := filepath.Abs(os.ExpandEnv(*root))

	archive := newArchive(backupName, rootPath, password)
	defer archive.Close()

	finder := inspection.NewFileFinder(rootPath)

	finder.Find(&archivingVisitor{
		archive: archive,
	})

	var handler archiving.MissingFileHandler

	switch *missingBehavior {
	case "restore":
		handler = RestoreMissingFileHandler()

	case "delete":
		handler = DeleteMissingFileHandler()

	default:
	case "noop":
		handler = NoopMissingFileHandler()
	}

	archive.HandleMissing(handler)
}

func readPassword() string {
	data, err := gopass.GetPasswdPrompt("Please enter your password: ", true, os.Stdin, os.Stdout)

	if nil != err {
		log.Panicf("Failed to read password: %v", err)
	}

	return string(data)
}
