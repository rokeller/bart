//go:build azure || azurite

package azureBlobs

import (
	"context"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/domain"
)

const (
	BLOBNAME_SETTINGS = "settings"
	BLOBNAME_INDEX    = "index"
)

type azureStorageProvider struct {
	client *container.Client
}

func NewAzuriteStorageProvider(containerName string) archiving.StorageProvider {
	connStr := "AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;DefaultEndpointsProtocol=http;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;QueueEndpoint=http://127.0.0.1:10001/devstoreaccount1;TableEndpoint=http://127.0.0.1:10002/devstoreaccount1;"
	blobClient, _ := azblob.NewClientFromConnectionString(connStr, nil)

	return newAzureStorageProvider(blobClient, containerName)
}

func NewAzureStorageProvider(
	serviceURL string,
	containerName string,
	cred azcore.TokenCredential,
) archiving.StorageProvider {
	blobClient, err := azblob.NewClient(serviceURL, cred, nil)
	if nil != err {
		glog.Fatalf("Failed to create Azure Blob client: %v", err)
	}

	return newAzureStorageProvider(blobClient, containerName)
}

func newAzureStorageProvider(blobClient *azblob.Client, containerName string) archiving.StorageProvider {
	containerClient := blobClient.ServiceClient().NewContainerClient(containerName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := containerClient.Create(ctx, nil)
	if nil != err {
		if !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
			glog.Fatalf("Failed to create target container: %v", err)
		}
	}

	return azureStorageProvider{
		client: containerClient,
	}
}

// DeleteBackupFile implements archiving.StorageProvider.
func (p azureStorageProvider) DeleteBackupFile(entry domain.Entry) error {
	panic("unimplemented")
}

// DeleteIndex implements archiving.StorageProvider.
func (p azureStorageProvider) DeleteIndex() error {
	panic("unimplemented")
}

// DeleteSettings implements archiving.StorageProvider.
func (p azureStorageProvider) DeleteSettings() error {
	panic("unimplemented")
}

// NewIndexWriter implements archiving.StorageProvider.
func (p azureStorageProvider) NewIndexWriter() (io.WriteCloser, error) {
	return p.newBlobWriter(BLOBNAME_INDEX)
}

// NewSettingsWriter implements archiving.StorageProvider.
func (p azureStorageProvider) NewSettingsWriter() (io.WriteCloser, error) {
	return p.newBlobWriter(BLOBNAME_SETTINGS)
}

// ReadBackupFile implements archiving.StorageProvider.
func (p azureStorageProvider) ReadBackupFile(entry domain.Entry) (io.ReadCloser, error) {
	blobName := blobNameForEntry(entry)
	// Not using a context with a timeout, since the index can be quite big
	// and take a while to read.
	return p.readBlob(blobName, nil)
}

// ReadIndex implements archiving.StorageProvider.
func (p azureStorageProvider) ReadIndex() (io.ReadCloser, error) {
	// Not using a context with a timeout, since the index can be quite big
	// and take a while to read.
	r, err := p.readBlob(BLOBNAME_INDEX, nil)
	if nil != err {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, archiving.IndexNotFound{}
		}

		return nil, err
	}

	return r, nil
}

// ReadSettings implements archiving.StorageProvider.
func (p azureStorageProvider) ReadSettings() (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r, err := p.readBlob(BLOBNAME_SETTINGS, ctx)
	if nil != err {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, archiving.SettingsNotFound{}
		}

		return nil, err
	}

	return r, nil
}

// WriteBackupFile implements archiving.StorageProvider.
func (p azureStorageProvider) WriteBackupFile(
	entry domain.Entry,
	file *os.File,
) error {
	blobName := blobNameForEntry(entry)

	return p.uploadFile(blobName, file, nil)
}

func (p azureStorageProvider) readBlob(blobName string, ctx context.Context) (io.ReadCloser, error) {
	if nil == ctx {
		ctx = context.Background()
	}

	blobClient := p.client.NewBlobClient(blobName)
	res, err := blobClient.DownloadStream(ctx, nil)
	if nil != err {
		return nil, err
	}

	return res.Body, nil
}

func (p azureStorageProvider) newBlobWriter(blobName string) (io.WriteCloser, error) {
	r, w := io.Pipe()

	wg := &sync.WaitGroup{}
	bw := blobWriteCloser{w: w, wg: wg}
	wg.Add(1)

	go func() {
		defer wg.Done()
		blobClient := p.client.NewBlockBlobClient(blobName)
		_, err := blobClient.UploadStream(context.Background(), r, nil)

		if nil != err {
			glog.Errorf("Failed to upload '%s': %v", blobName, err)
		} else {
			glog.Infof("Finished uploading '%s'.", blobName)
		}
	}()

	return bw, nil
}

func (p azureStorageProvider) uploadFile(
	blobName string,
	file *os.File,
	ctx context.Context) error {
	if nil == ctx {
		ctx = context.Background()
	}

	blobClient := p.client.NewBlockBlobClient(blobName)
	_, err := blobClient.UploadFile(ctx, file, nil)
	return err
}

func blobNameForEntry(entry domain.Entry) string {
	hash := entry.Hash()
	blobName := path.Join(hash[0:2], hash[2:4], hash)

	return blobName
}
