//go:build azure && !azurite

package main

import (
	"flag"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/golang/glog"
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
		glog.Fatal("The Azure blob service endpoint URL must not be empty.")
	}
}

func newStorageProvider(backupName string) archiving.StorageProvider {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		glog.Fatalf("Credentials for Azure could not be found: %v", err)
	}

	provider := azureBlobs.NewAzureStorageProvider(*serviceURL, backupName, cred)

	return provider
}
