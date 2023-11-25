package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
)

type cmdBase struct {
	args     commonArguments
	archive  archiving.Archive
	finished chan bool
}

type commonArguments struct {
	backupName          string
	localRoot           string
	degreeOfParallelism int
}

type Command interface {
	Run()
	Stop()
	Finished() <-chan bool
}

type commandFactory func([]string) Command

func parseCommand() Command {
	// The following is needed for glog, which puts its flags on the "shared" set.
	flag.Parse()
	allArgs := flag.Args()
	if len(allArgs) < 1 {
		glog.Exitln("Expected command 'backup', 'restore', or 'clean'.")
	}

	// Figure out what command we're dealing with first.
	var cmdFactory commandFactory
	switch strings.ToLower(allArgs[0]) {
	case "backup":
		cmdFactory = newBackupCommand
	case "restore":
		cmdFactory = newRestoreCommand
	case "clean":
		cmdFactory = newCleanCommand

	default:
		glog.Exitln("Expected command 'backup', 'restore', or 'clean'.")
	}

	cmd := cmdFactory(allArgs[1:])

	return cmd
}

func addCommonArgs(flagset *flag.FlagSet) *commonArguments {
	commonArgs := commonArguments{}
	flagset.StringVar(&commonArgs.backupName,
		"name", "backup", "The name of the backup archive.")
	flagset.StringVar(&commonArgs.localRoot,
		"path", ".", "The path to the directory to backup and/or restore.")
	flagset.IntVar(&commonArgs.degreeOfParallelism,
		"p", runtime.NumCPU(), "The degree of parallelism to use.")

	updateFlags(flagset)

	return &commonArgs
}

func newArchive(args commonArguments) archiving.Archive {
	args.backupName = strings.TrimSpace(args.backupName)

	if "" == strings.TrimSpace(args.backupName) {
		glog.Exit("The backup name must not be empty.")
	} else {
		verifyFlags()
	}

	password := readPassword()
	rootDir, _ := filepath.Abs(os.ExpandEnv(args.localRoot))
	localContext := archiving.NewLocalContext(rootDir)
	storageProvider := newStorageProvider(args.backupName)
	archive := archiving.NewArchive(password, localContext, storageProvider)

	return archive
}

func (c cmdBase) signalFinished() {
	c.finished <- true
}

func (c cmdBase) stop() {
	if err := c.archive.Close(); nil != err {
		glog.Errorf("Failed to close backup archive: %v", err)
	}
}
