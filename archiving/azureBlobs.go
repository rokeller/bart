//go:build !files

package archiving

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rokeller/bart/crypto"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

const (
	blobKeySettings = "settings"
	blobKeyIndex    = "index"
)

// AzureContext provides the context for operations in Azure.
type AzureContext struct {
	client *azblob.Client
}

// NewAzureContextFromTokenCredential creates a new context for managing archive blobs in Azure.
func NewAzureContextFromTokenCredential(serviceURL string, cred azcore.TokenCredential) AzureContext {
	client, _ := azblob.NewClient(serviceURL, cred, nil)

	return AzureContext{
		client: client,
	}
}

type azureArchiveContext struct {
	AzureContext

	container string
	context.Context
}

func (a AzureContext) newAzureArchiveContext(name string) azureArchiveContext {
	ctx := context.Background()

	return azureArchiveContext{
		AzureContext: a,
		container:    name,
		Context:      ctx,
	}.init()
}

func (a azureArchiveContext) init() azureArchiveContext {
	createOptions := &azblob.CreateContainerOptions{
		Metadata: map[string]*string{"owner": to.Ptr("bart")},
	}
	_, err := a.client.CreateContainer(a.Context, a.container, createOptions)

	if nil != err {
		if bloberror.HasCode(err, bloberror.ContainerAlreadyExists, bloberror.ResourceAlreadyExists) {
			log.Println("The container already exists.")
		} else {
			log.Panicf("The container could not be created: %v", err)
		}

		return a
	}

	log.Printf("The container was created.")

	return a
}

func (a *azureArchiveContext) getUploadWriter(blobKey string, accessTier blob.AccessTier) io.WriteCloser {
	var file *os.File
	var err error

	if file, err = os.CreateTemp(os.TempDir(), "bart-*"); nil != err {
		log.Panicf("Failed to create temp file: %v", err)
	}

	return &uploadWriter{
		azureArchiveContext: a,
		file:                file,
		path:                file.Name(),
		blobKey:             blobKey,
		accessTier:          accessTier,
	}
}

func (a *azureArchiveContext) uploadBlobFromFile(blobKey string, file *os.File, accessTier blob.AccessTier) {
	options := azblob.UploadFileOptions{
		AccessTier: &accessTier,
	}
	_, err := a.client.UploadFile(a.Context, a.container, blobKey, file, &options)
	if nil != err {
		log.Fatalf("Failed to upload blob '%s': %v", blobKey, err)
	}
}

func (a *azureArchiveContext) downloadBlob(blobKey string, failIfNotFound bool) io.ReadCloser {
	res, err := a.client.DownloadStream(a.Context, a.container, blobKey, nil)

	if nil != err {
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ResourceNotFound) {
			if failIfNotFound {
				log.Panicf("The blob was not found: %s.", blobKey)
			} else {
				return nil
			}
		} else {
			log.Panicf("The blob could not be downloaded; service response: %v", err)
		}
	}

	return res.Body
}

func (a *azureArchiveContext) removeBlob(blobKey string) {
	_, err := a.client.DeleteBlob(a.Context, a.container, blobKey, nil)

	if nil != err {
		log.Panicf("Failed to remove archive blob: %v", err)
	}
}

// ***** Settings *****

type azureSettings struct {
	azureArchiveContext
	*streamSettings
}

func newAzureSettings(archiveContext azureArchiveContext) Settings {
	return &azureSettings{
		azureArchiveContext: archiveContext,
		streamSettings: &streamSettings{
			settingsBase: &settingsBase{},
		},
	}
}

func (s *azureSettings) GetSalt() []byte {
	r := s.azureArchiveContext.downloadBlob(blobKeySettings, false)

	if nil != r {
		defer r.Close()

		s.loadSettings(r)
	} else {
		s.GenerateSalt()
	}

	defer s.uploadIfDirty()

	return s.salt
}

func (s *azureSettings) uploadIfDirty() {
	if !s.dirty {
		return
	}

	w := s.azureArchiveContext.getUploadWriter(blobKeySettings, blob.AccessTierHot)
	defer w.Close()

	s.storeSettings(w)
	s.dirty = false
}

// ***** Archive *****

type azureArchive struct {
	azureArchiveContext
	*cryptoArchive
}

// NewAzureArchive creates a new archive for backups in Azure.
func NewAzureArchive(azureContext AzureContext, localRoot, name, password string) Archive {
	archiveContext := azureContext.newAzureArchiveContext(name)
	settings := newAzureSettings(archiveContext)
	cryptoCtx := crypto.NewContext(password, settings.GetSalt())
	base := (&archiveBase{
		localRoot: localRoot,
		index:     newAzureIndex(cryptoCtx, archiveContext, name, true),
	}).init()

	return &azureArchive{
		azureArchiveContext: archiveContext,
		cryptoArchive: &cryptoArchive{
			archiveBase: base,
			Context:     cryptoCtx,
		},
	}
}

func (a *azureArchive) Backup(ctx inspection.Context) {
	if !a.shouldAddOrUpdate(ctx) {
		return
	}

	archiveBlobKey := a.getArchivedRelFilePath(ctx.RelPath())

	fmt.Printf("AddOrUpdate  %s  ->  %s", ctx.RelPath(), archiveBlobKey)
	fmt.Println()

	w := a.getUploadWriter(archiveBlobKey, blob.AccessTierCool)
	defer w.Close()

	a.backup(ctx, w)
}

func (a *azureArchive) Restore(entry domain.Entry) {
	archiveBlobKey := a.getArchivedRelFilePath(entry.RelPath)

	r := a.azureArchiveContext.downloadBlob(archiveBlobKey, true)

	defer r.Close()

	a.restore(entry, r)
}

func (a *azureArchive) Delete(entry domain.Entry) {
	archiveBlobKey := a.getArchivedRelFilePath(entry.RelPath)
	a.azureArchiveContext.removeBlob(archiveBlobKey)

	a.delete(entry)
}

func (a *azureArchive) HandleMissing(handler MissingFileHandler) {
	a.handleMissing(a, handler)
}

// ***** Index *****

type azureIndex struct {
	azureArchiveContext
	*cryptoIndex
}

func newAzureIndex(cryptoCtx *crypto.Context, archiveCtx azureArchiveContext, name string, compress bool) Index {
	return &azureIndex{
		azureArchiveContext: archiveCtx,
		cryptoIndex: &cryptoIndex{
			compress:    compress,
			streamIndex: &streamIndex{},
			Context:     cryptoCtx,
		},
	}
}

func (i *azureIndex) Load() {
	r := i.azureArchiveContext.downloadBlob(blobKeyIndex, false)

	if nil != r {
		defer r.Close()
	}

	cryptoReader := i.wrapReader(r)
	defer cryptoReader.Close()

	i.load(cryptoReader)
}

func (i *azureIndex) Store() {
	if i.dirty {
		w := i.azureArchiveContext.getUploadWriter(blobKeyIndex, blob.AccessTierHot)
		defer w.Close()

		cryptoWriter := i.wrapWriter(w)
		defer cryptoWriter.Close()

		i.store(cryptoWriter)
		log.Printf("Updated index.")
	}
}

type uploadWriter struct {
	*azureArchiveContext

	file       *os.File
	path       string
	blobKey    string
	accessTier blob.AccessTier
}

func (u *uploadWriter) Write(data []byte) (int, error) {
	return u.file.Write(data)
}

func (u *uploadWriter) Close() error {
	defer os.Remove(u.path)
	defer u.file.Close()

	// Upload the blob to Azure.
	u.azureArchiveContext.uploadBlobFromFile(u.blobKey, u.file, u.accessTier)

	return nil
}
