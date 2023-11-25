package archiving

import (
	"sync"

	"github.com/golang/glog"
	"github.com/rokeller/bart/domain"
)

type indexEntry struct {
	domain.EntryMetadata
	EntryFlags
}

type EntryFlags uint32

const (
	EntryFlagsNone            EntryFlags = 0x0000_0000
	EntryFlagsPresentInBackup EntryFlags = 0x0000_0001
	EntryFlagsPresentInLocal  EntryFlags = 0x0000_0002
)

type Index struct {
	archive *Archive

	// entries tracks entries in the index; must only accessed directly by
	// handleMessage or during initialization.
	entries map[string]indexEntry

	messages chan message
	dirty    bool
	closed   bool

	wgClose *sync.WaitGroup
}

func newIndex(a *Archive) *Index {
	index := Index{
		archive:  a,
		entries:  make(map[string]indexEntry),
		messages: make(chan message, 10),
		dirty:    false,
		closed:   false,

		wgClose: &sync.WaitGroup{},
	}

	index.load()
	index.wgClose.Add(1)
	go index.handleMessages()

	return &index
}

func (i *Index) Count() int {
	// TODO: we might need the message handler on the index to be stopped
	//       while we count the entries
	return len(i.entries)
}

func (i *Index) Dirty() bool {
	return i.dirty
}

func (i *Index) Close() error {
	close(i.messages)
	i.wgClose.Wait()
	i.closed = true

	if i.Dirty() {
		glog.Info("The archive index has changed and needs to be uploaded.")
		if err := i.writeIndex(); nil != err {
			glog.Errorf("The archive index could not be uploaded: %v", err)
			return err
		}
	} else {
		glog.Info("The archive index has not changed.")
	}

	return nil
}

func (i *Index) load() {
	if err := i.readIndex(); nil == err {
		return
	} else if err == IndexNotFound {
		// It's not an error if the index does not exist yet.
		return
	} else if err == IndexDecryptionFailed {
		glog.Exit("Index decryption failed. Did you provide the correct password?")
	} else {
		glog.Exit("Failed to load archive index: %v", err)
	}
}

func (i *Index) walkIndex(fn func(domain.Entry, EntryFlags) error) error {
	i.sync()

	// TODO: we probably need the message handler on the index to be stopped
	//       while we walk the entries map.
	for key, value := range i.entries {
		entry := domain.Entry{
			RelPath:       key,
			EntryMetadata: value.EntryMetadata,
		}

		if err := fn(entry, value.EntryFlags); nil != err {
			return err
		}
	}

	return nil
}

func (i *Index) needsBackup(entry domain.Entry) bool {
	indexEntry := i.getEntry(entry.RelPath)
	found := nil != indexEntry

	backupNeeded := !found ||
		(indexEntry.EntryFlags&EntryFlagsPresentInBackup) == EntryFlagsNone ||
		indexEntry.Timestamp < entry.Timestamp

	// Let's mark the file as present in local
	if found {
		i.setEntry(entry, indexEntry.EntryFlags|EntryFlagsPresentInLocal, false)
	}

	return backupNeeded
}
