package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/rpc"
	"github.com/sourcegraph/jsonrpc2"
)

type ConnectionState int

const (
	NotConnectedState ConnectionState = 0
	ConnectedState    ConnectionState = iota
	InitializedState  ConnectionState = iota
	ShutdownState     ConnectionState = iota
)

type Client struct {
	rpcConn   *jsonrpc2.Conn
	tcpConn   net.Conn
	addr      string
	processId int
	// handshake? bool
	connState   ConnectionState
	clientType  types.ClientType
	HandleFunc  func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request)
	OnReconnect func()
}

func (c *Client) processIdField() jsonrpc2.CallOption {
	// TODO: if !handshake { return nil }
	return jsonrpc2.ExtraField("processId", c.processId)
}

func (c *Client) EnsureConnection() error {
	if c.rpcConn != nil || c.connState != NotConnectedState {
		return nil
	}

	c.OnReconnect()
	if err := startDaemonProcess(); err != nil {
		log.Fatalln(err)
	}

	return c.Connect()
}

func (c *Client) Connect() error {
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

	c.connState = ConnectedState
	c.tcpConn = conn

	c.rpcConn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(&rpc.CustomStream{
			ReadCloser:  c.tcpConn,
			WriteCloser: c.tcpConn,
		}, jsonrpc2.VarintObjectCodec{}),
		jsonrpc2.AsyncHandler(c),
	)

	return c.Handshake()
}

func (c *Client) Close() error {
	if err := c.Shutdown(); err != nil {
		return err
	}
	return c.rpcConn.Close()
}

func (c *Client) Call(method types.Method, params any, result any) error {
	return c.rpcConn.Call(context.Background(), string(method), params, result, c.processIdField())
}

func (c *Client) Notify(method types.Method, params any) error {
	return c.rpcConn.Notify(context.Background(), string(method), params, c.processIdField())
}

func (c *Client) Handle(ctx context.Context, conn *jsonrpc2.Conn, r *jsonrpc2.Request) {
	c.HandleFunc(ctx, conn, r)
}

func (c *Client) Collect(err string) error {
	return c.Call(types.CollectMethod, err, nil)
}

func (c *Client) ResolveDocument(filepath string, content string) error {
	return c.Notify(types.ResolveDocumentMethod, map[string]any{
		"filepath": filepath,
		"content":  content,
	})
}

func (c *Client) UpdateDocument(filepath string, content string) error {
	return c.Notify(types.UpdateDocumentMethod, map[string]any{
		"filepath": filepath,
		"content":  content,
	})
}

func (c *Client) DeleteDocument(filepath string) error {
	return c.Notify(types.DeleteDocumentMethod, filepath)
}

func (c *Client) Handshake() error {
	err := c.Call(types.HandshakeMethod, &types.ClientInfo{
		ProcessId:  c.processId,
		ClientType: c.clientType,
	}, nil)

	if err != nil {
		return err
	}

	c.connState = InitializedState
	return c.Call(types.PingMethod, nil, nil)
}

func (c *Client) Shutdown() error {
	if c.connState == ShutdownState || c.connState == NotConnectedState {
		return nil
	}

	if err := c.Notify(types.ShutdownMethod, nil); err != nil {
		return err
	}

	c.connState = ShutdownState
	return nil
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

func NewClient(addr string, clientType types.ClientType, handlerFunc ...func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request)) *Client {
	cl := &Client{
		addr:       addr,
		rpcConn:    nil,
		processId:  os.Getpid(),
		clientType: clientType,
		connState:  NotConnectedState,
		HandleFunc: func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {},
		OnReconnect: func() {
			fmt.Println("> daemon not started. spawning...")
		},
	}

	if len(handlerFunc) > 0 {
		cl.HandleFunc = handlerFunc[0]
	}

	return cl
}

func Connect(addr string, clientType types.ClientType, handlerFunc ...func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request)) *Client {
	cl := NewClient(addr, clientType, handlerFunc...)
	cl.Connect()
	return cl
}
