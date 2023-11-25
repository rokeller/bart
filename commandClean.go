package main

import (
	"flag"

	"github.com/rokeller/bart/archiving"
)

type cmdClean struct {
	cmdBase
}

// C implements Command.
func (c *cmdClean) C() <-chan bool {
	return c.finished
}

// Run implements Command.
func (c *cmdClean) Run() {
	defer c.signalFinished()

	panic("unimplemented")
}

// Stop implements Command.
func (c *cmdClean) Stop() {
	c.stop()
}

func newCleanCommand(
	args []string,
	commonArgs commonArguments,
	archive archiving.Archive,
) Command {
	cleanCmd := updateUsage(flag.NewFlagSet("clean", flag.ExitOnError))
	cleanCmd.String("l", "backup", "The location to clean: 'backup' to remove "+
		"files missing locally from the backup, 'local' to remove files missing "+
		"in the backup from the local file system.")
	cleanCmd.Parse(args)

	return &cmdClean{
		cmdBase: cmdBase{
			args:     commonArgs,
			archive:  archive,
			finished: make(chan bool),
		},
	}
}
