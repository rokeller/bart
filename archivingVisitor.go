package main

import (
	"fmt"
	"io/fs"
	"log"
	"sync"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/domain"
)

type archivingVisitor struct {
	a     archiving.Archive
	wg    *sync.WaitGroup
	queue chan domain.Entry
}

func NewArchivingVisitor(a archiving.Archive, degreeOfParallelism int) archivingVisitor {
	v := archivingVisitor{
		a:     a,
		wg:    &sync.WaitGroup{},
		queue: make(chan domain.Entry, degreeOfParallelism*2),
	}

	for i := 0; i < degreeOfParallelism; i++ {
		v.wg.Add(1)
		go func(id int) {
			defer v.wg.Done()
			v.handleUploadQueue(id)
		}(i)
	}

	return v
}

func (v archivingVisitor) Complete() {
	close(v.queue)
	v.wg.Wait()
}

func (v archivingVisitor) VisitDir(path string, d fs.DirEntry) {
	// intentionally left blank
}

func (v archivingVisitor) VisitFile(path string, f fs.DirEntry) {
	info, err := f.Info()
	if nil != err {
		log.Printf("Couldn't get details of file '%s': %v", path, err)
		return
	}

	entry := domain.Entry{
		RelPath: path,
		EntryMetadata: domain.EntryMetadata{
			Timestamp: info.ModTime().Unix(),
		},
	}

	if v.a.NeedsBackup(entry) {
		v.queue <- entry
	}
}

// import (
// 	"log"
// 	"sync"
// 	"time"

// 	"github.com/rokeller/bart/archiving"
// 	"github.com/rokeller/bart/inspection"
// )

// func (v *archivingVisitor) Done() {
// 	close(v.queue)
// 	v.waitGroup.Wait()
// 	endTime := time.Now()
// 	duration := endTime.Sub(v.startTime)

// 	stats := v.archive.GetIndexStats()
// 	log.Printf("Inspection done in %v. Backup has %d files.",
// 		duration, stats.NumFiles)
// }

func (v archivingVisitor) handleUploadQueue(id int) {
	numSuccessful, numFailed := 0, 0

	for {
		entry, isOpen := <-v.queue
		if !isOpen {
			break
		}

		log.Printf("[Uploader-%d] Backup file '%s' ...", id, entry.RelPath)
		if err := v.a.Backup(entry); nil != err {
			numFailed++
			log.Printf("[Uploader-%d] Backup of file '%s' failed: %v", id, entry.RelPath, err)
		}
		numSuccessful++
		fmt.Println(entry.RelPath)
	}

	log.Printf("[Uploader-%d] Finished. Successfully backed up %d file(s), failed to backup %d file(s).",
		id, numSuccessful, numFailed)
}
