package archiving

import (
	"time"

	"github.com/golang/glog"
	"github.com/rokeller/bart/domain"
)

type message interface{}

type keyedMessage struct {
	relPath string
}

type setMessage struct {
	keyedMessage
	indexEntry
	markDirty bool
}

type getMessage struct {
	keyedMessage
	result chan<- *indexEntry
}

type delMessage struct {
	keyedMessage
}

type syncMessage struct {
	start chan<- bool // This channel is used to indicate that the synced logic can start
	done  <-chan bool // this channel is used to indicate that the synced logic is done
}

func (i Index) setEntry(entry domain.Entry, flags EntryFlags, markDirty bool) {
	if i.closed {
		glog.Warningf("Not sending 'set' message for '%s', because message handling has stopped.",
			entry.RelPath)
		return
	}

	i.messages <- setMessage{
		keyedMessage: keyedMessage{relPath: entry.RelPath},
		indexEntry: indexEntry{
			EntryMetadata: entry.EntryMetadata,
			EntryFlags:    flags,
		},
		markDirty: markDirty,
	}
}

func (i Index) getEntry(relPath string) *indexEntry {
	if i.closed {
		glog.Warningf("Not sending 'get' message for '%s', because message handling has stopped.",
			relPath)
		return nil
	}

	// Use an unbuffered channel: when the message handler has the response,
	// we need to be ready to accept it.
	resultChannel := make(chan *indexEntry)

	i.messages <- getMessage{
		keyedMessage: keyedMessage{relPath: relPath},
		result:       resultChannel,
	}

	return <-resultChannel
}

func (i Index) deleteEntry(relPath string) {
	if i.closed {
		glog.Warningf("Not sending 'delete' message for '%s', because message handling has stopped.",
			relPath)
		return
	}

	i.messages <- delMessage{
		keyedMessage: keyedMessage{relPath: relPath},
	}
}

func (i Index) sync(fn func()) {
	done := make(chan bool, 1)
	// Make sure the done signal always reaches the handler.
	defer func() {
		// Signal the message handler (if it's still running), that we're done.
		done <- true
	}()

	if !i.closed {
		start := make(chan bool, 1)
		sync := syncMessage{
			start: start,
			done:  done,
		}

		i.messages <- sync
		// Wait for the message handler to tell us that we can start
		<-start
	}

	fn()
}

func (i *Index) handleMessages() {
	// TODO: better ticker frequency -- should get from command line params
	maintenanceTicker := time.NewTicker(30 * time.Second)
	defer maintenanceTicker.Stop()

	for {
		select {
		case msg, hasMore := <-i.messages:
			if !hasMore {
				goto terminate
			}
			i.handleMessage(msg)

		case <-maintenanceTicker.C:
			glog.V(1).Info("MaintenaceTicker fired. Check for changes in index.")
			if i.dirty {
				glog.Info("The index has changed, upload current index checkpoint to backup destination.")
				// Calling writeIndex here is safe because message handling is
				// "paused" to handle the maintenance ticker.
				if err := i.writeIndex(); nil != err {
					glog.Errorf("The archive index could not be uploaded: %v", err)
				}
			}
		}
	}

terminate:
	glog.V(1).Info("Index message handling terminated.")
	i.wgClose.Done()
}

func (i *Index) handleMessage(msg message) {
	switch m := msg.(type) {
	case setMessage:
		if m.markDirty {
			i.dirty = true
		}
		i.entries[m.relPath] = m.indexEntry

	case getMessage:
		indexEntry, found := i.entries[m.relPath]
		if !found {
			m.result <- nil
		} else {
			m.result <- &indexEntry
		}

	case delMessage:
		_, found := i.entries[m.relPath]
		delete(i.entries, m.relPath)
		if found {
			// We removed an existing entry from the index, so mark it dirty.
			i.dirty = true
		}

	case syncMessage:
		// Signal the sender that its logic can start
		m.start <- true
		// Wait for the sender to have its logic finished.
		<-m.done
		close(m.start)

	default:
		glog.Warningf("Unsupported message type: %v", m)
	}
}
