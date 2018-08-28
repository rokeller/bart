// +build !files

package archiving

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/rokeller/bart/crypto"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"

	"github.com/Azure/azure-storage-blob-go/2018-03-28/azblob"
)

const (
	blobKeySettings = "settings"
	blobKeyIndex    = "index"

	mb            = 1024 * 1024
	maxBlockSize  = 100 * mb
	maxUploadSize = 256 * mb
)

// AzureContext provides the context for operations in Azure.
type AzureContext struct {
	credential azblob.Credential
	service    azblob.ServiceURL
}

// NewAzureContext creates a new context for managing archive blobs in Azure.
func NewAzureContext(accountName, accountKey string) AzureContext {
	cred := azblob.NewSharedKeyCredential(accountName, accountKey)
	pipe := azblob.NewPipeline(cred, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))

	return AzureContext{
		credential: cred,
		service:    azblob.NewServiceURL(*u, pipe),
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

func (a *azureArchiveContext) uploadBlobFromFile(blobKey string, file *os.File, len int64) {
	u := a.container.NewBlockBlobURL(blobKey)

	if len > maxUploadSize {
		// The file length exceeds the maximum allowed upload size, so upload blocks that then get committed together.
		a.uploadBlobBlocks(u, file, len)
	} else {
		// Upload the entire file with a single request.
		a.uploadBlob(u, file, len)
	}
}

func (a *azureArchiveContext) uploadBlob(blob azblob.BlockBlobURL, reader io.ReadSeeker, len int64) {
	u := a.getUploadPipeline(blob, len)

	// Upload the file with a single request.
	_, err := u.Upload(a.Context, reader, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})

	if nil != err {
		log.Panicf("Failed to upload blob: %v", err)
	}
}

func (a *azureArchiveContext) uploadBlobBlocks(blob azblob.BlockBlobURL, reader io.ReaderAt, len int64) {
	blockIds := make([]string, 0)

	for offset := int64(0); offset < len; offset += maxBlockSize {
		blockLen := len - offset

		if blockLen > maxBlockSize {
			blockLen = maxBlockSize
		}

		blockID := a.uploadBlock(blob, reader, offset, blockLen)
		blockIds = append(blockIds, blockID)
	}

	_, err := blob.CommitBlockList(a.Context, blockIds, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})

	if nil != err {
		log.Panicf("Failed to commit block blob list: %v", err)
	}
}

func (a *azureArchiveContext) uploadBlock(blob azblob.BlockBlobURL, reader io.ReaderAt, offset, len int64) string {
	u := a.getUploadPipeline(blob, len)
	blockID := getRandomBlockID()
	block := io.NewSectionReader(reader, offset, len)
	_, err := u.StageBlock(a.Context, blockID, block, azblob.LeaseAccessConditions{})

	if nil != err {
		log.Panicf("Failed to upload block %s of blob %s: %v", blockID, blob.String(), err)
	}

	return blockID
}

func (a *azureArchiveContext) getUploadPipeline(blob azblob.BlockBlobURL, len int64) azblob.BlockBlobURL {
	timeout := getRequestTimeout(len)
	u := blob.WithPipeline(azblob.NewPipeline(a.credential, azblob.PipelineOptions{
		Retry: azblob.RetryOptions{
			TryTimeout: timeout,
		},
	}))

	return u
}

func (a *azureArchiveContext) downloadBlob(blobKey string, failIfNotFound bool) io.ReadCloser {
	u := a.container.NewBlockBlobURL(blobKey)
	res, err := u.Download(a.Context, 0, 0, azblob.BlobAccessConditions{}, false)

	if nil != err {
		switch terr := err.(type) {
		case azblob.StorageError:
			switch terr.ServiceCode() { // Compare serviceCode to various ServiceCodeXxx constants
			case azblob.ServiceCodeBlobNotFound:
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

func (a *azureArchiveContext) removeBlob(blobKey string) {
	u := a.container.NewBlockBlobURL(blobKey)
	_, err := u.Delete(a.Context, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})

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

	archiveBlobKey := a.getArchivedRelFilePath(ctx.RelPath())

	fmt.Printf("AddOrUpdate  %s  ->  %s", ctx.RelPath(), archiveBlobKey)
	fmt.Println()

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
		w := i.azureArchiveContext.getUploadWriter(blobKeyIndex)
		defer w.Close()

		cryptoWriter := i.wrapWriter(w)
		defer cryptoWriter.Close()

		i.store(cryptoWriter)
		log.Printf("Updated index.")
	}
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

	// Get the length of the temp file.
	len, err := u.file.Seek(0, io.SeekCurrent)

	if nil != err {
		log.Panicf("Failed to determine length of file: %v", err)
	}

	// Seek back to the start of the temp file to prepare for the upload.
	u.file.Seek(0, io.SeekStart)

	// Upload the blob to Azure.
	u.azureArchiveContext.uploadBlobFromFile(u.blobKey, u.file, len)

	return nil
}

func getRequestTimeout(len int64) time.Duration {
	// Calculate the timeout to use from the size (in MB). We allow about 60 seconds per MB plus an additional 60 seconds.
	lenMb := int(len / mb)
	timeout := time.Duration(60+(60*lenMb)) * time.Second

	return timeout
}

func getRandomBlockID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); nil != err {
		log.Panicf("Failed to generate random block ID: %v", err)
	}

	return base64.StdEncoding.EncodeToString(b)
}
