//go:build files

package archiving

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/rokeller/bart/crypto"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
)

// ***** Settings *****

type fileSettings struct {
	path string

	*streamSettings
}

func newFileSettings(rootPath string) Settings {
	path := path.Join(rootPath, ".settings")

	return &fileSettings{
		path: path,
		streamSettings: &streamSettings{
			settingsBase: &settingsBase{},
		},
	}
}

func (s *fileSettings) GetSalt() []byte {
	file, err := os.Open(s.path)

	if os.IsNotExist(err) {
		s.GenerateSalt()
	} else if nil != err {
		log.Panicf("The settings file could not be opened: %v", err)
	} else {
		defer file.Close()
		s.loadSettings(file)
	}
	defer s.uploadIfDirty()

	return s.salt
}

func (s *fileSettings) uploadIfDirty() {
	if !s.dirty {
		return
	}

	file, err := os.Create(s.path)

	if nil != err {
		log.Panicf("Failed to create settings file: %v", err)
	}

	defer file.Close()

	s.storeSettings(file)
	s.dirty = false
}

// ***** Archive *****

type fsCryptoArchive struct {
	root string
	*cryptoArchive
}

// NewFileArchive creates a new file system based archive.
func NewFileArchive(archiveRoot, localRoot, name, password string) Archive {
	targetRoot := path.Join(archiveRoot, name)

	if err := os.MkdirAll(targetRoot, 0700); nil != err {
		log.Panicf("Failed to create directory: %v", err)
	}

	settings := newFileSettings(targetRoot)
	cryptoCtx := crypto.NewContext(password, settings.GetSalt())

	base := (&archiveBase{
		localRoot: localRoot,
		index:     newFileIndex(cryptoCtx, targetRoot, true),
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

	archiveFilePath := a.getArchivedFilePath(ctx.RelPath(), true)

	log.Printf("AddOrUpdate  %s  ->  %s", ctx.RelPath(), archiveFilePath)

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

func (a *fsCryptoArchive) Delete(entry domain.Entry) {
	a.removeArchiveFile(entry.RelPath)
	a.delete(entry)
}

func (a *fsCryptoArchive) HandleMissing(handler MissingFileHandler) {
	a.handleMissing(a, handler)
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

func (a *fsCryptoArchive) removeArchiveFile(relPath string) {
	archiveFilePath := a.getArchivedFilePath(relPath, false)

	if err := os.Remove(archiveFilePath); nil != err {
		log.Panicf("Failed to remove archive file: %v", err)
	}

	parent := path.Dir(archiveFilePath)

	if err := os.Remove(parent); nil == err {
		parent = path.Dir(parent)
		os.Remove(parent)
	}
}

// ***** Index *****

type fsCryptoIndex struct {
	path string

	*cryptoIndex
}

func newFileIndex(cryptoCtx *crypto.Context, rootPath string, compress bool) Index {
	var filePath string

	if compress {
		filePath = path.Join(rootPath, ".index.gz.encrypted")
	} else {
		filePath = path.Join(rootPath, ".index.encrypted")
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
	if i.dirty {
		w := i.getWriter()
		defer w.Close()

		i.store(w)
		log.Printf("Updated index.")
	}
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
