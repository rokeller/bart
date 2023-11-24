//go:build azure && !azurite

package main

import (
	"flag"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/providers/azureBlobs"
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

func newStorageProvider(backupName string) archiving.StorageProvider {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Credentials for Azure could not be found: %v", err)
	}

	provider := azureBlobs.NewAzureStorageProvider(*serviceURL, backupName, cred)

	return provider
}
