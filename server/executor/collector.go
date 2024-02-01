package executor

import (
	"fmt"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon/client"
)

type Collector interface {
	Collect(exitCode int, args, workingDir, stderr string) (r int, p int, err error)
}

type ClientCollector struct {
	*client.Client
}

func (cc *ClientCollector) Collect(exitCode int, args, workingDir, stderr string) (int, int, error) {
	r, p, err := cc.Client.Collect(exitCode, args, workingDir, stderr)
	if err != nil {
		fmt.Printf("[daemon-rpc|error] %s\n", err.Error())
	}

	if r > 0 {
		fmt.Fprintln(os.Stderr, stderr)
	}
	return r, p, err
}
