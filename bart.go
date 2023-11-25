package main

import (
	"os"
	"os/signal"

	"github.com/golang/glog"
	"github.com/howeyc/gopass"
)

func main() {
	cmd := parseCommand()
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
