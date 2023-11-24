package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/howeyc/gopass"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/domain"
	"github.com/rokeller/bart/inspection"
)

type commandContext struct {
	archive             archiving.Archive
	degreeOfParallelism int
	rootDir             string
	finished            chan bool
}

func main() {
	numCPU := runtime.NumCPU()

	name := flag.String("name", "backup", "The name of the backup archive.")
	root := flag.String("path", ".", "The path to the directory to backup and/or restore.")
	degreeOfParallelism := flag.Int("p", 2*numCPU, "The degree of parallelism to use.")
	missingBehavior := flag.String("m", "noop",
		"A behavior for files missing locally: 'noop' to do nothing, 'restore' "+
			"to restore them from the backup, 'delete' to delete them in the "+
			"backup archive.")

	updateFlags()
	flag.Parse()

	backupName := strings.TrimSpace(*name)

	if "" == backupName {
		glog.Fatalf("The backup name must not be empty.")
	} else {
		verifyFlags()
	}

	password := readPassword()
	rootDir, _ := filepath.Abs(os.ExpandEnv(*root))
	localContext := archiving.NewLocalContext(rootDir)
	storageProvider := newStorageProvider(backupName)
	archive := archiving.NewArchive(password, localContext, storageProvider)
	defer archive.Close()

	glog.V(2).Infof("missingBehavior: %v", *missingBehavior)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)

	ctx := commandContext{
		archive:             archive,
		degreeOfParallelism: *degreeOfParallelism,
		rootDir:             *root,
		finished:            make(chan bool),
	}

	go ctx.run()

	select {
	case s := <-c:
		glog.V(0).Info("Got signal:", s)
	case <-ctx.finished:
		break
	}
}

func (c commandContext) run() {
	// Visit local files and upload the ones missing or changed.
	visitor := NewArchivingVisitor(c.archive, c.degreeOfParallelism)
	err := inspection.Discover(c.rootDir, visitor)
	if nil != err {
		glog.Errorf("Discovery failed: %v", err)
	}
	visitor.Complete()

	// Find files that are missing locally.
	c.archive.FindLocallyMissing(func(entry domain.Entry) {
		// The item is present in the backup, but not locally.
		// TODO: decide based on command line actions
		glog.V(1).Infof("File '%s' is in backup, but not local.", entry.RelPath)
		if err := c.archive.Restore(entry); nil != err {
			glog.Errorf("failed to restore '%s': %v", entry.RelPath, err)
		} else {
			fmt.Println(entry.RelPath)
		}
	})

	c.finished <- true
}

func readPassword() string {
	data, err := gopass.GetPasswdPrompt("Please enter your password: ", true, os.Stdin, os.Stderr)

	if nil != err {
		glog.Fatalf("Failed to read password: %v", err)
	}

	return string(data)
}
