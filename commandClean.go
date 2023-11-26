package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
)

type CleanupLocation int

const (
	CleanupLocationBackup CleanupLocation = iota
	CleanupLocationLocal
)

type cmdCleanup struct {
	cmdBase

	location CleanupLocation
	wg       *sync.WaitGroup
	queue    chan deleteMessage
}

type deleteMessage interface{}

type deleteFromBackup struct {
	domain.Entry
}

type deleteFromLocal struct {
	relPath      string
	absolutePath string
}

// Finished implements Command.
func (c *cmdCleanup) Finished() <-chan bool {
	return c.finished
}

// Run implements Command.
func (c *cmdCleanup) Run() {
	defer c.signalFinished()

	for i := 0; i < c.args.degreeOfParallelism; i++ {
		c.wg.Add(1)
		go func(id int) {
			defer c.wg.Done()
			c.handleCleanupQueue(id)
		}(i)
	}

	switch c.location {
	case CleanupLocationBackup:
		c.cleanupBackup()
	case CleanupLocationLocal:
		c.cleanupLocal()

	default:
		glog.Fatalf("Unhandled cleanup location %d.", c.location)
	}
}

// Stop implements Command.
func (c *cmdCleanup) Stop() {
	close(c.queue)
	c.wg.Wait()

	c.stop()
}

func newCleanupCommand(args []string) Command {
	locationStr := "backup"
	cleanFlags := flag.NewFlagSet("clean", flag.ExitOnError)
	cleanFlags.StringVar(&locationStr, "l", "backup", "The location to clean: 'backup' to remove "+
		"files missing locally from the backup, 'local' to remove files missing "+
		"in the backup from the local file system.")
	commonArgs := addCommonArgs(cleanFlags)
	cleanFlags.Parse(args)

	var location CleanupLocation
	switch strings.ToLower(locationStr) {
	case "backup":
		location = CleanupLocationBackup
	case "local":
		location = CleanupLocationLocal

	default:
		glog.Exit("The cleanup location must either be 'backup' or 'local'.")
	}

	return &cmdCleanup{
		cmdBase: cmdBase{
			args:     *commonArgs,
			archive:  newArchive(*commonArgs),
			finished: make(chan bool),
		},

		location: location,
		wg:       &sync.WaitGroup{},
		queue:    make(chan deleteMessage, commonArgs.degreeOfParallelism*2),
	}
}

func (c *cmdCleanup) cleanupBackup() {
	// Find files that are in the backup index, but cannot be found locally and
	// queue their backup copy for deletion.
	c.archive.FindLocallyMissing(func(entry domain.Entry) {
		// The item is present in the backup, but not locally.
		absLocalPath := path.Join(c.args.localRoot, entry.RelPath)
		if glog.V(3) {
			glog.Infof("Checking local file '%s' ...", absLocalPath)
		}

		_, err := os.Stat(absLocalPath)
		if errors.Is(err, os.ErrNotExist) {
			if glog.V(3) {
				glog.Infof("Local file '%s' not found. Queue deletion of '%s' from backup",
					absLocalPath, entry.RelPath)
			}
			c.queue <- deleteFromBackup{Entry: entry}
		} else if nil != err {
			glog.Errorf("Failed to check for local file '%s': %v",
				entry.RelPath, err)
		}
	})
}

func (c *cmdCleanup) cleanupLocal() {
	// Find local files that are not in the backup and queue them for deletion
	// from the local file system.

	v := NewDeletingVisitor(c.archive, c.args.localRoot, c.queue)
	err := inspection.Discover(c.args.localRoot, v)
	if nil != err {
		glog.Errorf("Discovery failed: %v", err)
	}
}

func (c *cmdCleanup) handleCleanupQueue(id int) {
	numSuccessful, numFailed := 0, 0

	for {
		msg, isOpen := <-c.queue
		if !isOpen {
			break
		}

		switch m := msg.(type) {
		case deleteFromBackup:
			// Remove the entry from the backup, and from the backup index.
			glog.V(1).Infof("[Cleanup-%d] Remove file '%s' from backup ...",
				id, m.Entry.RelPath)
			if err := c.archive.Delete(m.Entry); nil != err {
				numFailed++
				glog.Errorf("[Cleanup-%d] Removal of file '%s' failed: %v",
					id, m.Entry.RelPath, err)
			} else {
				numSuccessful++
				fmt.Println(m.Entry.RelPath)
			}

		case deleteFromLocal:
			// Remove the file from the local file system.
			glog.V(1).Infof("[Cleanup-%d] Remove local file '%s' ...",
				id, m.relPath)
			if err := os.Remove(m.absolutePath); nil != err {
				numFailed++
				glog.Errorf("[Cleanup-%d] Removal of local file '%s' failed: %v",
					id, m.relPath, err)
			} else {
				numSuccessful++
				fmt.Println(m.relPath)
			}

		default:
			glog.Warningf("Unsupported message type: %v", m)
		}
	}

	glog.Infof("[Cleanup-%d] Finished. Successfully removed %d file(s), failed to remove %d file(s).",
		id, numSuccessful, numFailed)
}
