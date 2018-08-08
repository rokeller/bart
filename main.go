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
	restoreMissing := flag.Bool("m", false, "If set, restores files missing locally.")
	accountName := flag.String("acct", "", "The Azure Storage Account name.")
	accountKey := flag.String("key", "", "The Azure Storage Account Key.")
	flag.Parse()

	backupName := strings.TrimSpace(*name)

	if "" == backupName {
		log.Fatalf("The backup name must not be empty.")
	} else if "" == *accountName {
		log.Fatalf("The Azure Storage Account name (acct) must not be empty.")
	} else if "" == *accountKey {
		log.Fatalf("The Azure Storage Account key (key) must not be empty.")
	}

	password := readPassword()
	rootPath, _ := filepath.Abs(os.ExpandEnv(*root))
	log.Printf("Backup '%s' as '%s' to Azure.", rootPath, backupName)

	azCtx := archiving.NewAzureContext(*accountName, *accountKey)
	archive := archiving.NewAzureArchive(azCtx, rootPath, backupName, password)
	defer archive.Close()

	finder := inspection.NewFileFinder(rootPath)

	finder.Find(&archivingVisitor{
		archive: archive,
	})

	if *restoreMissing {
		archive.RestoreMissing()
	}

	archive.Close()
}

func readPassword() string {
	data, err := gopass.GetPasswdPrompt("Please enter your password: ", true, os.Stdin, os.Stdout)

	if nil != err {
		log.Panicf("Failed to read password: %v", err)
	}

	return string(data)
}
