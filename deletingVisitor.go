package main

import (
	"io/fs"
	"path"

	"github.com/rokeller/bart/archiving"
)

type deletingVisitor struct {
	a       archiving.Archive
	rootDir string
	queue   chan<- deleteMessage
}

func NewDeletingVisitor(a archiving.Archive, rootDir string, queue chan<- deleteMessage) deletingVisitor {
	v := deletingVisitor{
		a:       a,
		rootDir: rootDir,
		queue:   queue,
	}

	return v
}

func (v deletingVisitor) VisitDir(path string, d fs.DirEntry) {
	// intentionally left blank
}

func (v deletingVisitor) VisitFile(relPath string, f fs.DirEntry) {
	entry := v.a.GetEntry(relPath)
	if nil == entry {
		v.queue <- deleteFromLocal{
			relPath:      relPath,
			absolutePath: path.Join(v.rootDir, relPath),
		}
	}
}
