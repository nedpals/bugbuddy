package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type StderrMonitor struct {
	daemonClient *DaemonClient
	buf          bytes.Buffer
}

func (wr *StderrMonitor) Flush() {
	if wr.buf.Len() == 0 {
		return
	}

	if err := wr.daemonClient.Collect(wr.buf.String()); err != nil {
		fmt.Printf("[daemon-rpc|error] %s\n", err.Error())
	}

	wr.buf.Reset()
}

func (wr *StderrMonitor) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if p[0] == 9 {
		wr.buf.WriteByte('\n')
	} else if p[0] != 9 && wr.buf.Len() != 0 {
		wr.Flush()
	}

	return wr.buf.Write(p)
}

func monitorProcess(daemonClient *DaemonClient, prog string, args ...string) error {
	errProcessor := &StderrMonitor{daemonClient: daemonClient}
	if err := errProcessor.daemonClient.EnsureConnection(); err != nil {
		return err
	}

	defer func() {
		errProcessor.Flush()
		errProcessor.daemonClient.Close()
	}()

	fmt.Printf("> listening to %s %s...\n", prog, strings.Join(args, " "))
	progCmd := exec.Command(prog, args...)
	progCmd.Stdin = os.Stdin
	progCmd.Stdout = os.Stdout
	stderrPipe, err := progCmd.StderrPipe()
	if err != nil {
		return err
	} else if err := progCmd.Start(); err != nil {
		return err
	}

	sc := bufio.NewScanner(stderrPipe)
	for sc.Scan() {
		errProcessor.Write(sc.Bytes())
	}

	return progCmd.Wait()
}
