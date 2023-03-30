package main

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
)

type Daemon struct{}

func (d *Daemon) ProcessIncomingError(err string) error {
	return nil
}

func startDaemon(addr string) error {
	daemon := new(Daemon)
	rpc.Register(daemon)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return http.Serve(l, nil)
}

// TODO: add function ensuring daemon is alive. if not, spawn the daemon
func ensureDaemonAlive() error {
	return nil
}

// TODO:
func connectToDaemon(addr string) error {
	return nil
}

func listenToProcess(prog string, args ...string) error {
	fmt.Printf("> listening to %s %s...\n", prog, strings.Join(args, " "))

	progCmd := exec.Command(prog, args...)
	progCmd.Stdin = os.Stdin
	progCmd.Stdout = os.Stdout
	progCmd.Stderr = os.Stderr

	return progCmd.Run()
}
