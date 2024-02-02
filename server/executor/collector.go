package executor

import (
	"fmt"
	"log"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon/client"
)

type Collector interface {
	Collect(exitCode int, args, workingDir, stderr string) (r int, p int, err error)
}

type ClientCollector struct {
	Logger *log.Logger
	*client.Client
}

func (cc *ClientCollector) Collect(exitCode int, args, workingDir, stderr string) (int, int, error) {
	r, p, err := cc.Client.Collect(exitCode, args, workingDir, stderr)
	if err != nil {
		cc.Logger.Println(err)
	}

	if r > 0 {
		fmt.Fprintln(os.Stderr, stderr)
	}
	return r, p, err
}
