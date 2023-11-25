package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
)

type cmdBase struct {
	args     commonArguments
	archive  archiving.Archive
	finished chan bool
}

type Command interface {
	Run()
	Stop()
	C() <-chan bool
}

func parseCommand(args commonArguments) Command {
	remainingArgs := flag.Args()
	if len(remainingArgs) < 1 {
		glog.Exitln("Expected command 'backup', 'restore', or 'clean'.")
	}

	var cmdFactory func([]string, commonArguments, archiving.Archive) Command

	switch strings.ToLower(remainingArgs[0]) {
	case "backup":
		cmdFactory = newBackupCommand
	case "restore":
		cmdFactory = newRestoreCommand
	case "clean":
		cmdFactory = newCleanCommand

	default:
		glog.Exitln("Expected command 'backup', 'restore', or 'clean'.")
	}

	cmd := cmdFactory(remainingArgs[1:], args, newArchive(args))

	return cmd
}

func newArchive(args commonArguments) archiving.Archive {
	password := readPassword()
	rootDir, _ := filepath.Abs(os.ExpandEnv(args.localRoot))
	localContext := archiving.NewLocalContext(rootDir)
	storageProvider := newStorageProvider(args.backupName)
	archive := archiving.NewArchive(password, localContext, storageProvider)

	return archive
}

func updateUsage(flags *flag.FlagSet) *flag.FlagSet {
	oldUsage := flags.Usage
	flags.Usage = func() {
		oldUsage()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Common arguments:")
		flag.Usage()
	}

	return flags
}

func (c cmdBase) signalFinished() {
	c.finished <- true
}

func (c cmdBase) stop() {
	if err := c.archive.Close(); nil != err {
		glog.Errorf("Failed to close backup archive: %v", err)
	}
}
