package main

import (
	"fmt"

	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/domain"
)

// NoopMissingFileHandler creates a new noop missing file handler.
func NoopMissingFileHandler() archiving.MissingFileHandler {
	return &noopMissingFileHandler{}
}

type noopMissingFileHandler struct {
}

func (h *noopMissingFileHandler) HandleMissing(archive archiving.Archive, entry domain.Entry) {
	// Intentionally left blank.
}

// RestoreMissingFileHandler creates a restoring missing file handler.
func RestoreMissingFileHandler() archiving.MissingFileHandler {
	return &restoreMissingFileHandler{}
}

type restoreMissingFileHandler struct {
}

func (h *restoreMissingFileHandler) HandleMissing(archive archiving.Archive, entry domain.Entry) {
	archive.Restore(entry)
	fmt.Print("restored.")
}

// DeleteMissingFileHandler creates a deleting missing file handler.
func DeleteMissingFileHandler() archiving.MissingFileHandler {
	return &deleteMissingFileHandler{}
}

type deleteMissingFileHandler struct {
}

func (h *deleteMissingFileHandler) HandleMissing(archive archiving.Archive, entry domain.Entry) {
	archive.Delete(entry)
	fmt.Print("deleted from backup archive.")
}
