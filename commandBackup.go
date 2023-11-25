package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/rokeller/bart/archiving"
	"github.com/rokeller/bart/inspection"
)

type cmdBackup struct {
	cmdBase
}

// C implements Command.
func (c *cmdBackup) C() <-chan bool {
	return c.finished
}

// Run implements Command.
func (c *cmdBackup) Run() {
	defer c.signalFinished()

	// Visit local files and upload the ones missing or changed.
	visitor := NewArchivingVisitor(c.archive, c.args.degreeOfParallelism)
	err := inspection.Discover(c.args.localRoot, visitor)
	if nil != err {
		glog.Errorf("Discovery failed: %v", err)
	}
	visitor.Complete()
}

// Stop implements Command.
func (c *cmdBackup) Stop() {
	c.stop()
}

func newBackupCommand(
	args []string,
	commonArgs commonArguments,
	archive archiving.Archive,
) Command {
	backupFlags := updateUsage(flag.NewFlagSet("backup", flag.ExitOnError))
	backupFlags.Parse(args)

	return &cmdBackup{
		cmdBase: cmdBase{
			args:     commonArgs,
			archive:  archive,
			finished: make(chan bool),
		},
	}
}
