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

func updateFlags(flags *flag.FlagSet) {
	serviceURL = flags.String("azep", "", "The blob service endpoint URL.")
}

func verifyFlags() {
	if "" == *serviceURL {
		glog.Exit("The Azure blob service endpoint URL must not be empty.")
	}
}

func newStorageProvider(backupName string) archiving.StorageProvider {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		glog.Exit("Credentials for Azure could not be found: %v", err)
	}

	provider := azureBlobs.NewAzureStorageProvider(*serviceURL, backupName, cred)

	return provider
}
