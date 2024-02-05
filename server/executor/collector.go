package executor

import (
	"log"

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
	resp, err := cc.Client.Collect(exitCode, args, workingDir, stderr)
	if err != nil {
		cc.Logger.Println(err)
	}
	if resp == nil {
		return 0, 0, nil
	}
	return resp.Recognized, resp.Processed, err
}
