// +build !files

package main

import (
	"flag"
	"log"

	"github.com/rokeller/bart/archiving"
)

var (
	accountName *string
	accountKey  *string
)

func updateFlags() {
	accountName = flag.String("acct", "", "The Azure Storage Account name.")
	accountKey = flag.String("key", "", "The Azure Storage Account Key.")
}

func verifyFlags() {
	if "" == *accountName {
		log.Fatalf("The Azure Storage Account name (acct) must not be empty.")
	} else if "" == *accountKey {
		log.Fatalf("The Azure Storage Account key (key) must not be empty.")
	}
}

func newArchive(backupName, rootPath, password string) archiving.Archive {
	log.Printf("Backup '%s' as '%s' to Azure.", rootPath, backupName)
	azCtx := archiving.NewAzureContext(*accountName, *accountKey)
	archive := archiving.NewAzureArchive(azCtx, rootPath, backupName, password)

	return archive
}
