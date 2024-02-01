package executor

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type StderrMonitor struct {
	numErrors  int
	workingDir string
	exitCode   int
	args       []string
	collector  Collector
	buf        bytes.Buffer
}

func (wr *StderrMonitor) Flush() {
	if wr.buf.Len() == 0 {
		return
	}

	str := wr.buf.String()
	r, _, _ := wr.collector.Collect(wr.exitCode, strings.Join(wr.args, ""), wr.workingDir, str)
	wr.numErrors += r
	if r > 0 {
		wr.buf.Reset()
	}
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

func Execute(workingDir string, c Collector, prog string, args ...string) (int, int, error) {
	errProcessor := &StderrMonitor{
		workingDir: workingDir,
		collector:  c,
		args:       append([]string{prog}, args...),
		exitCode:   1,
	}
	defer errProcessor.Flush()

	progCmd := exec.Command(prog, args...)
	progCmd.Stdin = os.Stdin
	progCmd.Stdout = os.Stdout
	stderrPipe, _ := progCmd.StderrPipe()
	if err := progCmd.Start(); err != nil {
		return errProcessor.numErrors, 1, err
	}

	sc := bufio.NewScanner(stderrPipe)
	for sc.Scan() {
		errProcessor.Write(sc.Bytes())
		errProcessor.Flush()
	}

	if err, ok := progCmd.Wait().(*exec.ExitError); ok {
		errProcessor.exitCode = err.ExitCode()
		return errProcessor.numErrors, errProcessor.exitCode, nil
	}

	return errProcessor.numErrors, 0, nil
}
