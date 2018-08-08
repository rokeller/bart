package archiving

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/rokeller/bart/crypto"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
)

// ***** Archive *****

type fsCryptoArchive struct {
	root string
	*cryptoArchive
}

// NewFileArchive creates a new file system based archive.
func NewFileArchive(archiveRoot, localRoot, name string, cryptoCtx *crypto.Context) Archive {
	base := (&archiveBase{
		localRoot: localRoot,
		index:     newFileIndex(cryptoCtx, archiveRoot, name, true),
	}).init()

	return &fsCryptoArchive{
		root: path.Join(archiveRoot, name),
		cryptoArchive: &cryptoArchive{
			archiveBase: base,
			Context:     cryptoCtx,
		},
	}
}

func (a *fsCryptoArchive) Backup(ctx inspection.Context) {
	if !a.shouldAddOrUpdate(ctx) {
		return
	}

	fmt.Printf("AddOrUpdate  %s", ctx.RelPath())
	fmt.Println()

	archiveFilePath := a.getArchivedFilePath(ctx.RelPath(), true)
	outfile, err := os.Create(archiveFilePath)

	if nil != err {
		log.Panicf("Failed to create archive file: %v", err)
	}

	defer outfile.Close()

	a.backup(ctx, outfile)
}

func (a *fsCryptoArchive) Restore(entry domain.Entry) {
	archiveFilePath := a.getArchivedFilePath(entry.RelPath, false)
	infile, err := os.Open(archiveFilePath)

	if nil != err {
		log.Panicf("Failed to open file: %v", err)
	}

	defer infile.Close()

	a.restore(entry, infile)
}

func (a *fsCryptoArchive) RestoreMissing() {
	a.restoreMissing(a)
}

func (a *fsCryptoArchive) getArchivedFilePath(relPath string, createDirs bool) string {
	archivedRelPath := a.getArchivedRelFilePath(relPath)
	archivedRelDir := path.Dir(archivedRelPath)

	archiveDirPath := path.Join(a.root, archivedRelDir)

	if createDirs {
		if err := os.MkdirAll(archiveDirPath, 0700); nil != err {
			log.Panicf("Failed to create directory: %v", err)
		}
	}

	return path.Join(a.root, archivedRelPath)
}

// ***** Index *****

type fsCryptoIndex struct {
	path string

	*cryptoIndex
}

func newFileIndex(cryptoCtx *crypto.Context, basePath, name string, compress bool) Index {
	var filePath string

	if compress {
		filePath = path.Join(basePath, name+".gz.index")
	} else {
		filePath = path.Join(basePath, name+".index")
	}

	return &fsCryptoIndex{
		path: filePath,
		cryptoIndex: &cryptoIndex{
			compress:    compress,
			streamIndex: &streamIndex{},
			Context:     cryptoCtx,
		},
	}
}

func (i *fsCryptoIndex) Load() {
	r := i.getReader()
	defer r.Close()

	i.load(r)
}

func (i *fsCryptoIndex) Store() {
	w := i.getWriter()
	defer w.Close()

	i.store(w)
}

func (i *fsCryptoIndex) getReader() io.ReadCloser {
	file, err := os.Open(i.path)

	if os.IsNotExist(err) {
		// Return an empty reader.
		return ioutil.NopCloser(bytes.NewReader(make([]byte, 0)))
	} else if nil != err {
		log.Panicf("Failed to open backup index: %v", err)
	}

	return i.cryptoIndex.wrapReader(file)
}

func (i *fsCryptoIndex) getWriter() io.WriteCloser {
	file, err := os.Create(i.path)

	if nil != err {
		log.Panicf("Failed to create backup index: %v", err)
	}

	return i.cryptoIndex.wrapWriter(file)
}

type writeCloser struct {
	io.Writer
}

func (w writeCloser) Close() error {
	return nil
}
