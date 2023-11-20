package main

import (
	"log"
	"sync"
	"time"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/inspection"
)

type archivingVisitor struct {
	archive   archiving.Archive
	startTime time.Time

	degreeOfParallelism int
	queue               chan inspection.Context
	waitGroup           *sync.WaitGroup
}

func (v *archivingVisitor) Start() {
	v.queue = make(chan inspection.Context, 2*v.degreeOfParallelism)
	v.waitGroup = &sync.WaitGroup{}

	for i := 0; i < v.degreeOfParallelism; i++ {
		v.waitGroup.Add(1)
		go func(id int) {
			defer v.waitGroup.Done()
			v.uploadQueueHandler(id)
		}(i)
	}

	v.startTime = time.Now()
}

func (v *archivingVisitor) Visit(ctx inspection.Context) {
	if !ctx.Item.IsDir() {
		v.queue <- ctx
	}
}

func (v *archivingVisitor) Done() {
	close(v.queue)
	v.waitGroup.Wait()
	endTime := time.Now()
	duration := endTime.Sub(v.startTime)

	index := v.archive.GetBackupIndex()
	log.Printf("Inspection done in %v. Backup has %d files.", duration, len(index))
}

func (v *archivingVisitor) uploadQueueHandler(id int) {
	log.Printf("[Uploader-%d] Starting.", id)
	numHandled := 0

	for {
		ctx, isOpen := <-v.queue
		if !isOpen {
			break
		} else {
			v.archive.Backup(ctx)
			numHandled++
		}
	}

	log.Printf("[Uploader-%d] Finished; handled %d file(s).", id, numHandled)
}
