package archiving

import (
	"encoding/binary"
	"io"
	"log"

	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"

	"github.com/golang/protobuf/proto"
)

// ***** Settings *****

type streamSettings struct {
	*settingsBase
}

func (s *streamSettings) loadSettings(r io.Reader) {
	settingsSize := make([]byte, 4)
	_, err := io.ReadFull(r, settingsSize)

	if io.EOF == err {
		s.settingsBase.GenerateSalt()
		return
	} else if nil != err {
		log.Panicf("Failed to read settings size: %v", err)
	}

	dataSize := binary.LittleEndian.Uint32(settingsSize)
	data := make([]byte, dataSize)
	_, err = io.ReadFull(r, data)

	if nil != err {
		log.Panicf("Failed to read settings data: %v", err)
	}

	settings := &domain.Settings{}

	if err := proto.Unmarshal(data, settings); nil != err {
		log.Panicf("Failed to unmarschal settings: %v", err)
	}

	s.salt = settings.Salt
}

func (s *streamSettings) storeSettings(w io.Writer) {
	settings := &domain.Settings{
		Salt: s.salt,
	}

	data, err := proto.Marshal(settings)

	if nil != err {
		log.Panicf("Failed to marshal settings: %v", err)
	}

	settingsSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(settingsSize, uint32(len(data)))

	if _, err := w.Write(settingsSize); nil != err {
		log.Panicf("Failed to write settings size: %v", err)
	}

	if _, err := w.Write(data); nil != err {
		log.Panicf("Failed to write settings data: %v", err)
	}
}

// ***** Index *****

type streamIndex struct {
	index domain.BackupIndex

	entriesRead    int
	entriesWritten int
}

func (si *streamIndex) getIndex() domain.BackupIndex {
	return si.index
}

func (si *streamIndex) load(r io.Reader) {
	var entry *domain.Entry

	index := make(domain.BackupIndex)

	for {
		if entry = readEntry(r); nil == entry {
			break
		}

		index[entry.RelPath] = entry.EntryMetadata
		si.entriesRead++
	}

	si.index = index
}

func (si *streamIndex) store(w io.Writer) {
	for relPath, meta := range si.index {
		data := marshalEntry(relPath, meta)
		writeEntry(w, data)
		si.entriesWritten++
	}
}

func (si *streamIndex) shouldAddOrUpdate(ctx inspection.Context) bool {
	entry, found := si.index[ctx.RelPath()]

	return !found || entry.Timestamp < ctx.Timestamp()
}

func (si *streamIndex) AddOrUpdate(entry domain.Entry) {
	si.index[entry.RelPath] = entry.EntryMetadata
}

func (si *streamIndex) Remove(relPath string) {
	delete(si.index, relPath)
}

func (si *streamIndex) NumEntriesRead() int {
	return si.entriesRead
}

func (si *streamIndex) NumEntriesWritten() int {
	return si.entriesWritten
}

func readEntry(r io.Reader) *domain.Entry {
	entrySize := make([]byte, 4)
	_, err := io.ReadFull(r, entrySize)

	if io.EOF == err {
		return nil
	} else if nil != err {
		log.Panicf("Failed to read index entry size: %v", err)
	}

	dataSize := binary.LittleEndian.Uint32(entrySize)
	data := make([]byte, dataSize)
	_, err = io.ReadFull(r, data)

	if nil != err {
		log.Panicf("Failed to read index entry data: %v", err)
	}

	entry := &domain.IndexEntry{}

	if err := proto.Unmarshal(data, entry); nil != err {
		log.Panicf("Failed to unmarshal index entry: %v", err)
	}

	return &domain.Entry{
		RelPath: *entry.RelPath,
		EntryMetadata: domain.EntryMetadata{
			Timestamp: *entry.LastModified,
		},
	}
}

func writeEntry(w io.Writer, entry []byte) {
	entrySize := make([]byte, 4)
	binary.LittleEndian.PutUint32(entrySize, uint32(len(entry)))

	if _, err := w.Write(entrySize); nil != err {
		log.Panicf("Failed to write entry size: %v", err)
	}

	if _, err := w.Write(entry); nil != err {
		log.Panicf("Failed to write entry data: %v", err)
	}
}

func marshalEntry(relPath string, metadata domain.EntryMetadata) []byte {
	entry := &domain.IndexEntry{
		RelPath:      proto.String(relPath),
		LastModified: proto.Int64(metadata.Timestamp),
	}

	data, err := proto.Marshal(entry)

	if nil != err {
		log.Panicf("Failed to marshal entry: %v", err)
	}

	return data
}
