package main

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/golang/glog"
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
		glog.Errorf("Couldn't get details of file '%s': %v", path, err)
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

func (v archivingVisitor) handleUploadQueue(id int) {
	numSuccessful, numFailed := 0, 0

	for {
		entry, isOpen := <-v.queue
		if !isOpen {
			break
		}

		glog.V(1).Info("[Uploader-%d] Backup file '%s' ...", id, entry.RelPath)
		if err := v.a.Backup(entry); nil != err {
			numFailed++
			glog.Errorf("[Uploader-%d] Backup of file '%s' failed: %v", id, entry.RelPath, err)
		}
		numSuccessful++
		fmt.Println(entry.RelPath)
	}

	glog.Infof("[Uploader-%d] Finished. Successfully backed up %d file(s), failed to backup %d file(s).",
		id, numSuccessful, numFailed)
}
