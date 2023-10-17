//go:build !files

package main

import (
	"flag"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/rokeller/bart/archiving"
)

var (
	serviceURL *string
)

func updateFlags() {
	serviceURL = flag.String("azep", "", "The blob service endpoint URL.")
}

func verifyFlags() {
	if "" == *serviceURL {
		log.Fatalf("The Azure blob service endpoint URL must not be empty.")
	}
}

func newArchive(backupName, rootPath, password string) archiving.Archive {
	log.Printf("Backup '%s' as '%s' to Azure.", rootPath, backupName)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Credentials for Azure could not be found; error: %v", err)
	}

	azCtx := archiving.NewAzureContextFromTokenCredential(*serviceURL, cred)
	archive := archiving.NewAzureArchive(azCtx, rootPath, backupName, password)

	return archive
}
