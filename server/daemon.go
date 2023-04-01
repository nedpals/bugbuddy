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

type ClientType int

const (
	CLIENT_TYPE_MONITOR ClientType = 0
	CLIENT_TYPE_LSP     ClientType = iota
	CLIENT_TYPE_UNKNOWN ClientType = iota
)

type DaemonClientInfo struct {
	ProcessId  int        `json:"processId"`
	ClientType ClientType `json:"clientType"`
}

type DaemonServer struct {
	connectedClients map[int]ClientType
	errors           []string
}

// TODO: dummy payload for now. should give back instructions instead of the error message
type ErrorReport struct {
	Message string
}

func (d *DaemonServer) countLspClients() int {
	count := 0

	for _, typ := range d.connectedClients {
		if typ == CLIENT_TYPE_LSP {
			count++
		}
	}

	return count
}

func (d *DaemonServer) Collect(ctx context.Context, err string, c *jsonrpc2.Conn) (int, error) {
	fmt.Println(err)
	d.errors = append(d.errors, err)

	// TODO: process error first before notify
	fmt.Printf("> report new errors to %d clients\n", d.countLspClients())
	c.Notify(ctx, "clients/report", &ErrorReport{
		Message: err,
	})

	return 1, nil
}

func (d *DaemonServer) getProcessId(r *jsonrpc2.Request) int {
	for _, req := range r.ExtraFields {
		if req.Name != "processId" {
			continue
		} else if procId, ok := req.Value.(json.Number); ok {
			if num, err := procId.Int64(); err == nil {
				return int(num)
			} else {
				return -1
			}
		} else {
			return -2
		}
	}
	return -1
}

func (d *DaemonServer) checkProcessConnection(r *jsonrpc2.Request) *jsonrpc2.Error {
	procId := d.getProcessId(r)
	if procId == -2 {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Invalid process ID",
		}
	} else if _, found := d.connectedClients[procId]; !found {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process not connected yet.",
		}
	} else if procId == -1 {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process ID not found",
		}
	}

	return nil
}

func (d *DaemonServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	if r.Method != "handshake" && r.Method != "disconnect" {
		if err := d.checkProcessConnection(r); err != nil {
			c.ReplyWithError(ctx, r.ID, err)
			return
		}
	}

	switch r.Method {
	case "handshake":
		// TODO: add checks and result
		var info DaemonClientInfo
		if err := json.Unmarshal(*r.Params, &info); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		} else if info.ClientType < CLIENT_TYPE_MONITOR || info.ClientType >= CLIENT_TYPE_UNKNOWN {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unknown client type.",
			})
			return
		}

		fmt.Printf("> connected: {process_id: %d, type: %d}\n", info.ProcessId, info.ClientType)
		d.connectedClients[info.ProcessId] = info.ClientType
		c.Reply(ctx, r.ID, 1)
	case "shutdown":
		procId := d.getProcessId(r)
		delete(d.connectedClients, procId)
		fmt.Printf("> disconnected: {process_id: %d}\n", procId)
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
	return startRpcServer(addr, jsonrpc2.VarintObjectCodec{}, &DaemonServer{
		connectedClients: map[int]ClientType{},
		errors:           []string{},
	})
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

type ClientConnectionState int

const (
	CLIENT_NOT_CONNECTED ClientConnectionState = 0
	CLIENT_CONNECTED     ClientConnectionState = iota
	CLIENT_INITIALIZED   ClientConnectionState = iota
	CLIENT_SHUTDOWN      ClientConnectionState = iota
)

type DaemonClient struct {
	rpcConn   *jsonrpc2.Conn
	tcpConn   net.Conn
	addr      string
	processId int
	// handshake? bool
	connState   ClientConnectionState
	clientType  ClientType
	HandleFunc  func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request)
	OnReconnect func()
}

func (c *DaemonClient) processIdField() jsonrpc2.CallOption {
	// TODO: if !handshake { return nil }
	return jsonrpc2.ExtraField("processId", c.processId)
}

func (c *DaemonClient) Call(method string, params any, result any) error {
	return c.rpcConn.Call(context.Background(), method, params, result, c.processIdField())
}

func (c *DaemonClient) Notify(method string, params any) error {
	return c.rpcConn.Notify(context.Background(), method, params, c.processIdField())
}

func (c *DaemonClient) Handle(ctx context.Context, conn *jsonrpc2.Conn, r *jsonrpc2.Request) {
	c.HandleFunc(ctx, conn, r)
}

func (c *DaemonClient) Collect(err string) error {
	return c.Call("collect", err, nil)
}

func (c *DaemonClient) Handshake() error {
	err := c.Call("handshake", &DaemonClientInfo{
		ProcessId:  c.processId,
		ClientType: c.clientType,
	}, nil)

	if err != nil {
		return err
	}

	c.connState = CLIENT_INITIALIZED
	return nil
}

func (c *DaemonClient) Close() error {
	if err := c.Shutdown(); err != nil {
		return err
	}
	return c.rpcConn.Close()
}

func (c *DaemonClient) Shutdown() error {
	if c.connState == CLIENT_SHUTDOWN || c.connState == CLIENT_NOT_CONNECTED {
		return nil
	}

	if err := c.Notify("shutdown", nil); err != nil {
		return err
	}

	c.connState = CLIENT_SHUTDOWN
	return nil
}

func (c *DaemonClient) EnsureConnection() error {
	if c.rpcConn != nil || c.connState == CLIENT_NOT_CONNECTED {
		return nil
	}

	c.OnReconnect()
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

	if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
		return err
	}

	if err := conn.(*net.TCPConn).SetKeepAlivePeriod(10 * time.Second); err != nil {
		return err
	}

	c.connState = CLIENT_CONNECTED
	c.tcpConn = conn

	c.rpcConn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(c.tcpConn, jsonrpc2.VarintObjectCodec{}),
		c,
	)

	return c.Handshake()
}

func connectToDaemon(addr string, clientType ClientType, handlerFunc ...func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request)) *DaemonClient {
	cl := &DaemonClient{
		addr:       addr,
		rpcConn:    nil,
		processId:  os.Getpid(),
		clientType: clientType,
		connState:  CLIENT_NOT_CONNECTED,
		HandleFunc: func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {},
		OnReconnect: func() {
			fmt.Println("> daemon not started. spawning...")
		},
	}

	if len(handlerFunc) > 0 {
		cl.HandleFunc = handlerFunc[0]
	}

	cl.Connect()
	return cl
}
