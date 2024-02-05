package executor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var DefaultFprintWr io.Writer = os.Stderr

type StderrMonitor struct {
	numErrors  int
	workingDir string
	exitCode   int
	args       []string
	fPrintWr   io.Writer
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
	n, err = wr.buf.Write(p)
	if err != nil {
		return
	}
	fmt.Fprint(wr.fPrintWr, wr.buf.String())
	return
}

type executionLogicalOp int

const (
	ExecutionLogicalOpNone executionLogicalOp = 0
	ExecutionLogicalOpOr   executionLogicalOp = iota
	ExecutionLogicalOpAnd  executionLogicalOp = iota
)

var logicalOps = map[string]executionLogicalOp{
	"||": ExecutionLogicalOpOr,
	"&&": ExecutionLogicalOpAnd,
}

func hasLogicalOpInCommand(cmd string) string {
	for op := range logicalOps {
		if strings.Contains(cmd, op) {
			return op
		}
	}
	return ""
}

func Execute(workingDir string, c Collector, prog string, args ...string) (int, int, error) {
	sep := hasLogicalOpInCommand(prog)
	if sep != "" {
		commands := strings.Split(prog, sep)
		var err error
		var numErrors int
		var exitCode int
		for i, cmd := range commands {
			argv := strings.Split(strings.TrimSpace(cmd), " ")
			numErrors, exitCode, err = Execute(workingDir, c, argv[0], argv[1:]...)
			if err != nil {
				return numErrors, exitCode, err
			}

			if i < len(commands)-1 {
				switch logicalOps[sep] {
				case ExecutionLogicalOpOr:
					if exitCode == 0 {
						continue
					}
				case ExecutionLogicalOpAnd:
					if exitCode != 0 {
						continue
					}
				}
			}
		}
		return numErrors, exitCode, nil
	}

	if strings.Count(prog, " ") > 0 {
		splt := strings.Split(prog, " ")
		prog = splt[0]
		args = append(splt[1:], args...)
	}

	errProcessor := &StderrMonitor{
		workingDir: workingDir,
		collector:  c,
		args:       append([]string{prog}, args...),
		exitCode:   0,
		fPrintWr:   DefaultFprintWr,
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
		// if len(sc.Bytes()) == 0 {
		// 	errProcessor.Flush()
		// 	continue
		// }
		errProcessor.Write(sc.Bytes())
	}

	if err, ok := progCmd.Wait().(*exec.ExitError); ok {
		errProcessor.exitCode = err.ExitCode()
	}

	// flush remaining errors to collector
	if len(errProcessor.buf.Bytes()) > 0 {
		errProcessor.Flush()
	} else if errProcessor.exitCode == 0 {
		// collect immediately
		_, _, _ = errProcessor.collector.Collect(errProcessor.exitCode, strings.Join(errProcessor.args, ""), errProcessor.workingDir, "")
	}

	return errProcessor.numErrors, errProcessor.exitCode, nil
}
