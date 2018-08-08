package archiving

import (
	"backup/crypto"
	"backup/domain"
	"backup/inspection"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/Azure/azure-storage-blob-go/2018-03-28/azblob"
)

const (
	blobKeySettings = "settings"
	blobKeyIndex    = "index"
)

// AzureContext provides the context for operations in Azure.
type AzureContext struct {
	service azblob.ServiceURL
}

// NewAzureContext creates a new context for managing archive blobs in Azure.
func NewAzureContext(accountName, accountKey string) AzureContext {
	cred := azblob.NewSharedKeyCredential(accountName, accountKey)
	pipe := azblob.NewPipeline(cred, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))

	return AzureContext{
		service: azblob.NewServiceURL(*u, pipe),
	}
}

type azureArchiveContext struct {
	AzureContext
	container azblob.ContainerURL
	context.Context
}

func (a AzureContext) newAzureArchiveContext(name string) azureArchiveContext {
	u := a.service.NewContainerURL(name)
	ctx := context.Background()

	return azureArchiveContext{
		AzureContext: a,
		container:    u,
		Context:      ctx,
	}.init()
}

func (a azureArchiveContext) init() azureArchiveContext {
	_, err := a.container.Create(a.Context, azblob.Metadata{}, azblob.PublicAccessNone)

	if nil != err {
		switch terr := err.(type) {
		case azblob.StorageError:
			switch terr.ServiceCode() { // Compare serviceCode to various ServiceCodeXxx constants
			case azblob.ServiceCodeContainerAlreadyExists:
				log.Println("The container already exists.")
				return a

			default:
				log.Panicf("The container could not be created, service response: %v", terr)
			}
		default:
			log.Panicf("The container could not be created: %v", err)
		}

		return a
	}

	log.Printf("The container was created.")

	return a
}

func (a *azureArchiveContext) getUploadWriter(blobKey string) io.WriteCloser {
	var file *os.File
	var err error

	if file, err = ioutil.TempFile(os.TempDir(), "backup"); nil != err {
		log.Panicf("Failed to create temp file: %v", err)
	}

	return &uploadWriter{
		azureArchiveContext: a,
		file:                file,
		path:                file.Name(),
		blobKey:             blobKey,
	}
}

func (a *azureArchiveContext) uploadBlobFromReader(blobKey string, rs io.ReadSeeker) {
	u := a.container.NewBlockBlobURL(blobKey)
	_, err := u.Upload(a.Context, rs, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})

	if nil != err {
		log.Panicf("Failed to upload blob: %v", err)
	}
}

func (a *azureArchiveContext) downloadBlob(blobKey string, failIfNotFound bool) io.ReadCloser {
	u := a.container.NewBlockBlobURL(blobKey)
	res, err := u.Download(a.Context, 0, 0, azblob.BlobAccessConditions{}, false)

	if nil != err {
		switch terr := err.(type) {
		case azblob.StorageError:
			switch terr.ServiceCode() { // Compare serviceCode to various ServiceCodeXxx constants
			case azblob.ServiceCodeBlobNotFound:
				log.Println("The blob does not exist.")
				if failIfNotFound {
					log.Panicf("The blob was not found: %v", err)
				} else {
					return nil
				}

			default:
				log.Panicf("The blob could not be downloaded, service response: %v", terr)
			}
		default:
			log.Panicf("The blob could not be downloaded: %v", err)
		}
	}

	return res.Body(azblob.RetryReaderOptions{})
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

	w := s.azureArchiveContext.getUploadWriter(blobKeySettings)
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

	fmt.Printf("AddOrUpdate  %s", ctx.RelPath())
	fmt.Println()

	archiveBlobKey := a.getArchivedRelFilePath(ctx.RelPath())

	w := a.getUploadWriter(archiveBlobKey)
	defer w.Close()

	a.backup(ctx, w)
}

func (a *azureArchive) Restore(entry domain.Entry) {
	archiveBlobKey := a.getArchivedRelFilePath(entry.RelPath)

	r := a.azureArchiveContext.downloadBlob(archiveBlobKey, true)

	defer r.Close()

	a.restore(entry, r)
}

func (a *azureArchive) RestoreMissing() {
	a.restoreMissing(a)
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
	w := i.azureArchiveContext.getUploadWriter(blobKeyIndex)
	defer w.Close()

	cryptoWriter := i.wrapWriter(w)
	defer cryptoWriter.Close()

	i.store(cryptoWriter)
}

type uploadWriter struct {
	*azureArchiveContext

	file    *os.File
	path    string
	blobKey string
}

func (u *uploadWriter) Write(data []byte) (int, error) {
	return u.file.Write(data)
}

func (u *uploadWriter) Close() error {
	defer os.Remove(u.path)
	defer u.file.Close()

	// Seek back to the start of the temp file to prepare for the upload.
	u.file.Seek(0, io.SeekStart)

	// Upload the blob to Azure.
	u.azureArchiveContext.uploadBlobFromReader(u.blobKey, u.file)

	return nil
}
