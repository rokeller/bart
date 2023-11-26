package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/golang/glog"
	"github.com/rokeller/bart/domain"
)

type cmdRestore struct {
	cmdBase

	wg    *sync.WaitGroup
	queue chan domain.Entry
}

// Finished implements Command.
func (c *cmdRestore) Finished() <-chan bool {
	return c.finished
}

// Run implements Command.
func (c *cmdRestore) Run() {
	defer c.signalFinished()

	for i := 0; i < c.args.degreeOfParallelism; i++ {
		c.wg.Add(1)
		go func(id int) {
			defer c.wg.Done()
			c.handleRestoreQueue(id)
		}(i)
	}

	// Find files that are missing locally.
	c.archive.FindLocallyMissing(func(entry domain.Entry) {
		// The item is present in the backup, but not locally.
		absLocalPath := path.Join(c.args.localRoot, entry.RelPath)
		_, err := os.Stat(absLocalPath)
		if errors.Is(err, os.ErrNotExist) {
			c.queue <- entry
		} else if nil != err {
			glog.Errorf("Failed to check for local file '%s': %v",
				entry.RelPath, err)
		}
	})
}

// Stop implements Command.
func (c *cmdRestore) Stop() {
	close(c.queue)
	c.wg.Wait()

	c.stop()
}

func newRestoreCommand(args []string) Command {
	restoreFlags := flag.NewFlagSet("restore", flag.ExitOnError)
	commonArgs := addCommonArgs(restoreFlags)
	restoreFlags.Parse(args)

	return &cmdRestore{
		cmdBase: cmdBase{
			args:     *commonArgs,
			archive:  newArchive(*commonArgs),
			finished: make(chan bool),
		},

		wg:    &sync.WaitGroup{},
		queue: make(chan domain.Entry, commonArgs.degreeOfParallelism*2),
	}
}

func (c *cmdRestore) handleRestoreQueue(id int) {
	numSuccessful, numFailed := 0, 0

	for {
		entry, isOpen := <-c.queue
		if !isOpen {
			break
		}

		glog.V(1).Infof("[Restorer-%d] Restoring file '%s' ...",
			id, entry.RelPath)

		if err := c.archive.Restore(entry); nil != err {
			numFailed++
			glog.Errorf("[Restorer-%d] Restore of file '%s' failed: %v",
				id, entry.RelPath, err)
		} else {
			numSuccessful++
			fmt.Println(entry.RelPath)
		}
	}

	glog.Infof("[Restore-%d] Finished. Successfully restored %d file(s), failed to restore %d file(s).",
		id, numSuccessful, numFailed)
}
