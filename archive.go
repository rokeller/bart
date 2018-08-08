package main

import (
	"backup/archiving"
	"backup/inspection"
	"log"
	"time"
)

type archivingVisitor struct {
	archive   archiving.Archive
	startTime time.Time
}

func (v *archivingVisitor) Start() {
	v.startTime = time.Now()
}

func (v *archivingVisitor) Visit(ctx inspection.Context) {
	if !ctx.Item.IsDir() {
		v.archive.Backup(ctx)
	}
}

func (v *archivingVisitor) Done() {
	endTime := time.Now()
	duration := endTime.Sub(v.startTime)

	log.Printf("Inspection done in %v", duration)
}
