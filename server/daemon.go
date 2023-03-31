package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

const DEFAULT_DAEMON_PORT = ":3434"

type DaemonServer struct {
	errors []string
}

// TODO: dummy payload for now. should give back instructions instead of the error message
type ErrorReport struct {
	Message string
}

func (d *DaemonServer) Collect(ctx context.Context, err string, c *jsonrpc2.Conn) (int, error) {
	fmt.Println(err)
	d.errors = append(d.errors, err)

	// TODO: process error first before notify
	c.Notify(ctx, "clients/report", &ErrorReport{
		Message: err,
	})

	return 1, nil
}

func (d *DaemonServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	switch r.Method {
	case "collect":
		var errorStr string
		if err := json.Unmarshal(*r.Params, &errorStr); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		}

		n, err := d.Collect(ctx, errorStr, c)
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
		} else {
			c.Reply(ctx, r.ID, n)
		}
	}
}

func startDaemon(addr string) error {
	fmt.Println("> daemon started on " + addr)
	return startRpcServer(addr, jsonrpc2.VarintObjectCodec{}, &DaemonServer{})
}

// TODO: add function ensuring daemon is alive. if not, spawn the daemon
func startDaemonProcess() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// TODO: kill existing daemon process if found
	cmd := exec.Command(execPath, "daemon")
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Process.Release(); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

type DaemonClient struct {
	*jsonrpc2.Conn
	tcpConn net.Conn
	addr    string
}

func (c *DaemonClient) Handle(ctx context.Context, conn *jsonrpc2.Conn, r *jsonrpc2.Request) {}

func (c *DaemonClient) Collect(err string) error {
	return c.Call(context.Background(), "collect", err, nil)
}

func (c *DaemonClient) EnsureConnection() error {
	if c.Conn != nil {
		return nil
	}

	fmt.Println("> daemon not started. spawning...")
	if err := startDaemonProcess(); err != nil {
		log.Fatalln(err)
	}

	return c.Connect()
}

func (c *DaemonClient) Connect() error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}

	c.tcpConn = conn
	c.Conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(c.tcpConn, jsonrpc2.VarintObjectCodec{}),
		c,
	)

	return nil
}

func connectToDaemon(addr string) *DaemonClient {
	cl := &DaemonClient{
		addr: addr,
		Conn: nil,
	}

	cl.Connect()
	return cl
}
