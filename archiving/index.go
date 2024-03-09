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
	// handleMessages, handleMessage or during initialization.
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
	var count int
	i.sync(func() {
		count = len(i.entries)
	})

	return count
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
		// Calling writeIndex here is safe because message handler must have
		// been stopped at the beginning of the method.
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

// walkIndex walks through the index. It is the caller's responsibility to make
// sure there is mutually exclusive access to the index, e.g. through the use
// of Index.sync, or by calling before index message handling has started or
// after it has finished.
func (i *Index) walkIndex(fn func(domain.Entry, EntryFlags) error) error {
	var err error
	err = nil

	for key, value := range i.entries {
		entry := domain.Entry{
			RelPath:       key,
			EntryMetadata: value.EntryMetadata,
		}

		if err = fn(entry, value.EntryFlags); nil != err {
			break
		}
	}

	return err
}

// walkIndexSnapshot creates a snapshot of the current index and then walks the
// entries of that snapshot, applying the given fn for every entry.
func (i *Index) walkIndexSnapshot(fn func(domain.Entry, EntryFlags) error) error {
	// Track a snapshot of the current index' keys.
	snapshot := make(map[string]indexEntry)
	extractKeys := func() {
		for key, value := range i.entries {
			snapshot[key] = value
		}
	}
	i.sync(extractKeys)

	// Now iterate through the snapshot applying the fn to every item.
	var err error
	err = nil

	for key, value := range snapshot {
		entry := domain.Entry{
			RelPath:       key,
			EntryMetadata: value.EntryMetadata,
		}

		if err = fn(entry, value.EntryFlags); nil != err {
			break
		}
	}

	return err
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
