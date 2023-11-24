package archiving

import (
	"log"
	"time"

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

type syncMessage = chan<- bool

func (i Index) setEntry(entry domain.Entry, flags EntryFlags, markDirty bool) {
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
	resultChannel := make(chan *indexEntry)

	i.messages <- getMessage{
		keyedMessage: keyedMessage{relPath: relPath},
		result:       resultChannel,
	}

	return <-resultChannel
}

func (i Index) deleteEntry(relPath string) {
	i.messages <- delMessage{
		keyedMessage: keyedMessage{relPath: relPath},
	}
}

func (i *Index) handleMessages() {
	maintenanceTicker := time.NewTicker(5 * time.Second)
	defer maintenanceTicker.Stop()

	for {
		select {
		case msg, hasMore := <-i.messages:
			if !hasMore {
				goto terminate
			}
			i.handleMessage(msg)

		case <-maintenanceTicker.C:
			log.Printf("Time for maintenance: upload current index to backup destination")
		}
	}

terminate:
	log.Println("Index message handling terminated.")
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
		i.dirty = true
		delete(i.entries, m.relPath)

	case syncMessage:
		m <- true
		close(m)

	default:
		log.Printf("Unsupported message type: %v", m)
	}
}
