package main

import (
	"flag"
)

type cmdClean struct {
	cmdBase
}

// C implements Command.
func (c *cmdClean) Finished() <-chan bool {
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

func newCleanCommand(args []string) Command {
	cleanFlags := flag.NewFlagSet("clean", flag.ExitOnError)
	cleanFlags.String("l", "backup", "The location to clean: 'backup' to remove "+
		"files missing locally from the backup, 'local' to remove files missing "+
		"in the backup from the local file system.")
	commonArgs := addCommonArgs(cleanFlags)
	cleanFlags.Parse(args)

	return &cmdClean{
		cmdBase: cmdBase{
			args:     *commonArgs,
			archive:  newArchive(*commonArgs),
			finished: make(chan bool),
		},
	}
}
