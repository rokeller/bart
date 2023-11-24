//go:build azurite || (all && !azure && !files)

package main

import (
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/providers/azureBlobs"
)

func updateFlags() {
}

func verifyFlags() {
}

func newStorageProvider(backupName string) archiving.StorageProvider {
	provider := azureBlobs.NewAzuriteStorageProvider(backupName)

	return provider
}
