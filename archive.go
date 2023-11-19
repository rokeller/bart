package main

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/inspection"
)

type archivingVisitor struct {
	archive   archiving.Archive
	startTime time.Time
	queue     chan inspection.Context
	waitGroup *sync.WaitGroup
}

func (v *archivingVisitor) Start() {
	numCPU := runtime.NumCPU()
	v.queue = make(chan inspection.Context, numCPU*4)
	v.waitGroup = &sync.WaitGroup{}

	for i := 0; i < numCPU; i++ {
		v.waitGroup.Add(1)
		go func(id int) {
			defer v.waitGroup.Done()
			v.queueHandler(id)
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

func (v *archivingVisitor) queueHandler(id int) {
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
