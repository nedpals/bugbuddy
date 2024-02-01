package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
	"github.com/sourcegraph/jsonrpc2"
)

type ConnectionState int

const (
	NotConnectedState ConnectionState = 0
	ShutdownState     ConnectionState = iota
	ConnectedState    ConnectionState = iota
	InitializedState  ConnectionState = iota
)

const defaultConnectDelay = 750 * time.Millisecond

type MaxConnRetriesReachedError struct {
	Err error
}

func (e *MaxConnRetriesReachedError) Error() string {
	return fmt.Sprintf("max connection retries reached: %s", e.Err.Error())
}

type Client struct {
	context             context.Context
	rpcConn             *jsonrpc2.Conn
	tcpConn             net.Conn
	connRetries         int
	addr                string
	processId           int
	connState           ConnectionState
	clientType          types.ClientType
	HandleFunc          rpc.HandlerFunc
	SpawnOnMaxReconnect bool
	OnReconnect         func(int, error) bool
	OnSpawnDaemon       func()
}

func (c *Client) SetId(id int) {
	c.processId = id
}

func (c *Client) processIdField() jsonrpc2.CallOption {
	// TODO: if !handshake { return nil }
	if c.processId < 0 {
		return nil
	}
	return jsonrpc2.ExtraField("processId", c.processId)
}

func (c *Client) IsConnected() bool {
	return c.rpcConn != nil && c.connState >= ConnectedState
}

func (c *Client) tryReconnect(reason error) error {
	if c.connState != NotConnectedState {
		return nil
	}

	c.connRetries++
	if shouldReconnect := c.OnReconnect(c.connRetries, reason); !shouldReconnect {
		if c.SpawnOnMaxReconnect {
			c.OnSpawnDaemon()
			if err := startDaemonProcess(); err != nil {
				return err
			}

			// avoid looping
			c.SpawnOnMaxReconnect = false
			time.Sleep(defaultConnectDelay)
			if err := c.Connect(); err != nil {
				return err
			}

			// revert to original state if connection is successful
			c.SpawnOnMaxReconnect = true

			// this is important or else the below code will
			// interpret this as if it is was not able to reach to
			// the daemon server
			return nil
		}
		return &MaxConnRetriesReachedError{reason}
	}

	time.Sleep(defaultConnectDelay)
	return c.Connect()
}

func (c *Client) SetConn(conn net.Conn) {
	c.tcpConn = conn
}

func (c *Client) Connect() error {
	if c.context == nil {
		c.context = context.Background()
	}

	if c.tcpConn == nil {
		conn, err := net.Dial("tcp", c.addr)
		if err != nil {
			if err, ok := err.(*net.OpError); ok {
				if strings.HasSuffix(err.Err.Error(), "connection refused") {
					return c.tryReconnect(err)
				}
			}
			return err
		}

		if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
			return err
		}

		if err := conn.(*net.TCPConn).SetKeepAlivePeriod(10 * time.Second); err != nil {
			return err
		}

		c.SetConn(conn)
	}

	c.connState = ConnectedState
	c.connRetries = 0

	c.rpcConn = jsonrpc2.NewConn(
		c.context,
		jsonrpc2.NewBufferedStream(&rpc.CustomStream{
			ReadCloser:  c.tcpConn,
			WriteCloser: c.tcpConn,
		}, jsonrpc2.VarintObjectCodec{}),
		jsonrpc2.AsyncHandler(c),
	)

	return c.Handshake()
}

func (c *Client) Close() error {
	if c == nil || c.rpcConn == nil {
		return nil
	}
	if err := c.Shutdown(); err != nil {
		return err
	}
	return c.rpcConn.Close()
}

func (c *Client) Call(method types.Method, params any, result any) error {
	err := c.rpcConn.Call(c.context, string(method), params, result, c.processIdField())
	if err == jsonrpc2.ErrClosed {
		c.connState = NotConnectedState
		if err := c.tryReconnect(err); err != nil {
			return err
		}

		// retry again
		return c.Call(method, params, result)
	}
	return err
}

func (c *Client) Notify(method types.Method, params any) error {
	err := c.rpcConn.Notify(c.context, string(method), params, c.processIdField())
	if err == jsonrpc2.ErrClosed {
		c.connState = NotConnectedState
		if err := c.tryReconnect(err); err != nil {
			return err
		}

		// retry again
		return c.Notify(method, params)
	}
	return err
}

func (c *Client) Handle(ctx context.Context, conn *jsonrpc2.Conn, r *jsonrpc2.Request) {
	c.HandleFunc.Handle(ctx, conn, r)
}

func (c *Client) GenerateParticipantId() (string, error) {
	var gotParticipantId string
	if err := c.Call(types.GenerateParticipantIdMethod, nil, &gotParticipantId); err != nil {
		return "", err
	}
	return gotParticipantId, nil
}

func (c *Client) RetrieveParticipantId() (string, error) {
	var gotParticipantId string
	if err := c.Call(types.RetrieveParticipantIdMethod, nil, &gotParticipantId); err != nil {
		return "", err
	}
	return gotParticipantId, nil
}

func (c *Client) ResetLogger() error {
	return c.Call(types.ResetLoggerMethod, nil, nil)
}

func (c *Client) Collect(errCode int, command, workingDir, errMsg string) (int, int, error) {
	response := map[string]any{}
	recognizedErrors := 0
	processedErrors := 0
	err := c.Call(types.CollectMethod, types.CollectPayload{
		ErrorCode:  errCode,
		Command:    command,
		Error:      errMsg,
		WorkingDir: workingDir,
	}, &response)
	if rawRErrors, ok := response["recognized"]; ok {
		if rErrors, ok := rawRErrors.(float64); ok {
			recognizedErrors = int(rErrors)
		}
	}
	if rawPErrors, ok := response["processed"]; ok {
		if pErrors, ok := rawPErrors.(float64); ok {
			processedErrors = int(pErrors)
		}
	}
	if cErr, ok := response["error"].(string); ok && len(cErr) > 0 {
		err = fmt.Errorf(cErr)
	}
	return recognizedErrors, processedErrors, err
}

func (c *Client) ResolveDocument(filepath string, content string) error {
	return c.Call(types.ResolveDocumentMethod, types.DocumentPayload{
		DocumentIdentifier: types.DocumentIdentifier{Filepath: filepath},
		Content:            content,
	}, nil)
}

func (c *Client) UpdateDocument(filepath string, content string) error {
	return c.Call(types.UpdateDocumentMethod, types.DocumentPayload{
		DocumentIdentifier: types.DocumentIdentifier{Filepath: filepath},
		Content:            content,
	}, nil)
}

func (c *Client) DeleteDocument(filepath string) error {
	return c.Call(types.DeleteDocumentMethod, types.DocumentIdentifier{
		Filepath: filepath,
	}, nil)
}

func (c *Client) Handshake() error {
	var result int
	err := c.Call(types.HandshakeMethod, &types.ClientInfo{
		ProcessId:  c.processId,
		ClientType: c.clientType,
	}, &result)

	if err != nil {
		return err
	} else if result != 1 {
		return fmt.Errorf("failed to handshake with daemon server")
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

	time.Sleep(defaultConnectDelay)
	return nil
}

func NewClient(ctx context.Context, addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *Client {
	cl := &Client{
		addr:        addr,
		rpcConn:     nil,
		processId:   os.Getpid(),
		clientType:  clientType,
		connState:   NotConnectedState,
		HandleFunc:  func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {},
		OnReconnect: func(retries int, _ error) bool { return retries < 5 },
		OnSpawnDaemon: func() {
			fmt.Println("> daemon not started. spawning...")
		},
	}

	if len(handlerFunc) > 0 {
		cl.HandleFunc = handlerFunc[0]
	}

	return cl
}

func Connect(addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *Client {
	cl := NewClient(context.Background(), addr, clientType, handlerFunc...)
	cl.Connect()
	return cl
}
