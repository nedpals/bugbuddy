package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"time"
)

const DEFAULT_DAEMON_PORT = ":3434"

type DaemonServer struct {
	errors []string
}

func (d *DaemonServer) Collect(err string, reply *int) error {
	fmt.Println(err)

	d.errors = append(d.errors, err)
	*reply = 1

	return nil
}

func startDaemon(addr string) error {
	daemon := new(DaemonServer)
	rpc.Register(daemon)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Println("> daemon started on " + addr)
	return http.Serve(l, nil)
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
	*rpc.Client
	addr string
}

func (c *DaemonClient) Collect(err string) error {
	return c.Call("DaemonServer.Collect", err, nil)
}

func (c *DaemonClient) EnsureConnection() error {
	if c.Client != nil {
		return nil
	}

	fmt.Println("> daemon not started. spawning...")
	if err := startDaemonProcess(); err != nil {
		log.Fatalln(err)
	}

	return c.Connect()
}

func (c *DaemonClient) Connect() error {
	client, err := rpc.DialHTTP("tcp", c.addr)
	if err != nil {
		return err
	}

	c.Client = client
	return nil
}

func connectToDaemon(addr string) *DaemonClient {
	cl := &DaemonClient{
		addr:   addr,
		Client: nil,
	}

	cl.Connect()
	return cl
}
