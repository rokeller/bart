package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/howeyc/gopass"
)

type commonArguments struct {
	backupName          string
	localRoot           string
	degreeOfParallelism int
}

func main() {
	numCPU := runtime.NumCPU()
	args := commonArguments{}
	flag.StringVar(&args.backupName, "name", "backup", "The name of the backup archive.")
	flag.StringVar(&args.localRoot, "path", ".", "The path to the directory to backup and/or restore.")
	flag.IntVar(&args.degreeOfParallelism, "p", 2*numCPU, "The degree of parallelism to use.")

	updateFlags()
	flag.Parse()

	backupName := strings.TrimSpace(args.backupName)

	if "" == backupName {
		glog.Exit("The backup name must not be empty.")
	} else {
		verifyFlags()
	}

	cmd := parseCommand(args)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go cmd.Run()
	defer cmd.Stop()

	select {
	case s := <-c:
		glog.V(0).Info("Got signal:", s)
	case <-cmd.C():
		// The command has finished by itself.
		break
	}
}

func readPassword() string {
	data, err := gopass.GetPasswdPrompt("Please enter your password: ", true, os.Stdin, os.Stderr)

	if nil != err {
		glog.Exitf("Failed to read password: %v", err)
	}

	return string(data)
}
