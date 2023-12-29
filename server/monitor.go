package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	daemonClient "github.com/nedpals/bugbuddy/server/daemon/client"
)

type StderrMonitor struct {
	numErrors    int
	workingDir   string
	exitCode     int
	args         []string
	daemonClient *daemonClient.Client
	buf          bytes.Buffer
}

func (wr *StderrMonitor) Flush() {
	if wr.buf.Len() == 0 {
		return
	}

	if err := wr.daemonClient.Collect(wr.exitCode, strings.Join(wr.args, ""), wr.workingDir, wr.buf.String()); err != nil {
		fmt.Printf("[daemon-rpc|error] %s\n", err.Error())
	}

	os.Stderr.Write(wr.buf.Bytes())
	wr.numErrors++
	wr.buf.Reset()
}

func (wr *StderrMonitor) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if wr.buf.Len() != 0 {
		wr.buf.WriteByte('\n')
	}
	return wr.buf.Write(p)
}

func monitorProcess(workingDir string, daemonClient *daemonClient.Client, prog string, args ...string) (int, int, error) {
	errProcessor := &StderrMonitor{
		workingDir:   workingDir,
		daemonClient: daemonClient,
		args:         append([]string{prog}, args...),
		exitCode:     1,
	}
	defer errProcessor.Flush()

	fmt.Printf("> listening to %s %s...\n", prog, strings.Join(args, " "))
	progCmd := exec.Command(prog, args...)
	progCmd.Stdin = os.Stdin
	progCmd.Stdout = os.Stdout
	stderrPipe, err := progCmd.StderrPipe()
	if err != nil {
		return errProcessor.numErrors, 1, err
	} else if err := progCmd.Start(); err != nil {
		return errProcessor.numErrors, 1, err
	}

	sc := bufio.NewScanner(stderrPipe)
	for sc.Scan() {
		errProcessor.Write(sc.Bytes())
	}

	if err, ok := progCmd.Wait().(*exec.ExitError); ok {
		errProcessor.exitCode = err.ExitCode()
		return errProcessor.numErrors, errProcessor.exitCode, nil
	}

	return errProcessor.numErrors, 0, nil
}
