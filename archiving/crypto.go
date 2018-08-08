package archiving

import (
	"backup/crypto"
	"backup/domain"
	"backup/inspection"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
)

type cryptoArchive struct {
	*archiveBase
	*crypto.Context
}

func (a *cryptoArchive) backup(ctx inspection.Context, archiveWriter io.Writer) {
	encryptingWriter := crypto.NewCryptoWriter(a.Context, archiveWriter)

	a.archiveBase.backup(ctx, encryptingWriter)
}

func (a *cryptoArchive) restore(entry domain.Entry, archiveReader io.Reader) {
	decryptingReader := crypto.NewCryptoReader(a.Context, archiveReader)

	a.archiveBase.restore(entry, decryptingReader)
}

type cryptoIndex struct {
	compress bool

	*streamIndex
	*crypto.Context
}

func (i *cryptoIndex) wrapReader(r io.Reader) io.ReadCloser {
	if nil == r {
		// Return an empty reader.
		return ioutil.NopCloser(bytes.NewReader(make([]byte, 0)))
	}

	// Decrypt the file for reading.
	decryptingReader := crypto.NewCryptoReader(i.Context, r)
	var finalReader io.ReadCloser
	var err error

	if i.compress {
		// Wrap a gzip reader around the decrypted stream.
		finalReader, err = gzip.NewReader(decryptingReader)

		if nil != err {
			log.Panicf("Failed to create gzip reader: %v", err)
		}
	} else {
		finalReader = ioutil.NopCloser(decryptingReader)
	}

	return finalReader
}

func (i *cryptoIndex) wrapWriter(w io.Writer) io.WriteCloser {
	encryptingWriter := crypto.NewCryptoWriter(i.Context, w)

	var finalWriter io.WriteCloser

	if i.compress {
		// Also wrap a gzip writer around the input file.
		finalWriter = gzip.NewWriter(encryptingWriter)
	} else {
		finalWriter = writeCloser{encryptingWriter}
	}

	return finalWriter
}
