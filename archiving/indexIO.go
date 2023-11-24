package archiving

import (
	"compress/gzip"
	"encoding/binary"
	"io"

	"github.com/golang/glog"
	"github.com/rokeller/bart/domain"
	"google.golang.org/protobuf/proto"
)

func (i *Index) readIndex() error {
	r, err := i.archive.storageProvider.ReadIndex()
	if nil != err {
		glog.Errorf("error reading index from provider: %v", err)
		return err
	}
	defer r.Close()

	// Decrypt the stream holding the index ...
	cr, err := i.archive.cryptoContext.Decrypt(r)
	if nil != err {
		glog.Errorf("error decrypting index from provider: %v", err)
		return err
	}

	// ... and decompress it.
	gr, err := gzip.NewReader(cr)
	if nil != err {
		glog.Errorf("error decompressing index from provider: %v", err)
		return err
	}
	defer gr.Close()

	for {
		entry, err := readIndexEntry(gr)
		if nil != err {
			return err
		} else if nil == entry {
			break
		}

		// We do this outside of the entries map message handler because it
		// happens during startup and the handler doesn't need to be running
		// yet.
		i.entries[entry.RelPath] = indexEntry{
			EntryMetadata: entry.EntryMetadata,
			EntryFlags:    EntryFlagsPresentInBackup,
		}
	}

	return nil
}

func (i *Index) writeIndex() error {
	w, err := i.archive.storageProvider.NewIndexWriter()
	if nil != err {
		return err
	}
	defer w.Close()

	// ... and then encrypt it.
	cw, err := i.archive.cryptoContext.Encrypt(w)
	if nil != err {
		return err
	}
	defer cw.Close()

	// Compress the data in the index ...
	gw := gzip.NewWriter(cw)
	defer gw.Close()

	numEntries := 0
	err = i.walkIndex(func(e domain.Entry, ef EntryFlags) error {
		numEntries++
		return writeIndexEntry(e, gw)
	})

	glog.Infof("Archive index with %d file(s) uploaded.", numEntries)

	return err
}

func readIndexEntry(r io.Reader) (*domain.Entry, error) {
	entrySize := make([]byte, 4)

	if _, err := io.ReadFull(r, entrySize); io.EOF == err {
		return nil, nil
	} else if nil != err {
		return nil, err
	}

	dataSize := binary.LittleEndian.Uint32(entrySize)
	data := make([]byte, dataSize)
	if _, err := io.ReadFull(r, data); nil != err {
		return nil, err
	}

	entry := &domain.IndexEntry{}

	if err := proto.Unmarshal(data, entry); nil != err {
		return nil, err
	}

	return &domain.Entry{
		RelPath: *entry.RelPath,
		EntryMetadata: domain.EntryMetadata{
			Timestamp: *entry.LastModified,
		},
	}, nil
}

func writeIndexEntry(e domain.Entry, w io.Writer) error {
	buffer, err := marshalEntry(e)
	if nil != err {
		return err
	}

	entrySize := make([]byte, 4)
	binary.LittleEndian.PutUint32(entrySize, uint32(len(buffer)))

	if _, err := w.Write(entrySize); nil != err {
		return err
	}

	if _, err := w.Write(buffer); nil != err {
		return err
	}

	return nil
}

func marshalEntry(e domain.Entry) ([]byte, error) {
	entry := &domain.IndexEntry{
		RelPath:      proto.String(e.RelPath),
		LastModified: proto.Int64(e.Timestamp),
	}

	data, err := proto.Marshal(entry)

	if nil != err {
		return nil, err
	}

	return data, nil
}
