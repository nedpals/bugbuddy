package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/error_analyzer"
	"github.com/nedpals/bugbuddy-proto/server/rpc"
	"github.com/nedpals/bugbuddy-proto/server/store"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	store            store.Store
	connectedClients map[int]types.ClientType
	errors           []string
}

func (d *Server) countLspClients() int {
	count := 0

	for _, typ := range d.connectedClients {
		if typ == types.LspClientType {
			count++
		}
	}

	return count
}

func (d *Server) getProcessId(r *jsonrpc2.Request) int {
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

func (d *Server) checkProcessConnection(r *jsonrpc2.Request) *jsonrpc2.Error {
	procId := d.getProcessId(r)

	if _, found := d.connectedClients[procId]; !found {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process not connected yet.",
		}
	} else if procId == -2 {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Invalid process ID",
		}
	} else if procId == -1 {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process ID not found",
		}
	}

	return nil
}

func (d *Server) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	if !types.MethodIsEither(r.Method, types.HandshakeMethod, types.ShutdownMethod) {
		if err := d.checkProcessConnection(r); err != nil {
			c.ReplyWithError(ctx, r.ID, err)
			return
		}
	}

	switch types.Method(r.Method) {
	case types.HandshakeMethod:
		// TODO: add checks and result
		var info types.ClientInfo
		if err := json.Unmarshal(*r.Params, &info); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		} else if info.ClientType < types.MonitorClientType || info.ClientType >= types.UnknownClientType {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unknown client type.",
			})
			return
		}

		fmt.Printf("> connected: {process_id: %d, type: %d}\n", info.ProcessId, info.ClientType)
		d.connectedClients[info.ProcessId] = info.ClientType
		c.Reply(ctx, r.ID, 1)
	case types.ShutdownMethod:
		procId := d.getProcessId(r)
		delete(d.connectedClients, procId)
		fmt.Printf("> disconnected: {process_id: %d}\n", procId)
	case types.CollectMethod:
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
	case types.PingMethod:
		procId := d.getProcessId(r)
		fmt.Printf("> ping from %d\n", procId)
	}
}

func (d *Server) Collect(ctx context.Context, err string, c *jsonrpc2.Conn) (int, error) {
	fmt.Println(err)
	d.errors = append(d.errors, err)

	// TODO: process error first before notify
	fmt.Printf("> report new errors to %d clients\n", d.countLspClients())
	go analyzeAndSendError(ctx, err, c)

	return 1, nil
}

func Start(addr string) error {
	fmt.Println("> daemon started on " + addr)
	return rpc.StartServer(addr, jsonrpc2.VarintObjectCodec{}, &Server{
		connectedClients: map[int]types.ClientType{},
		errors:           []string{},
	})
}

func analyzeAndSendError(ctx context.Context, err string, c *jsonrpc2.Conn) {
	suggestion := error_analyzer.Default.Analyze(err)
	c.Notify(ctx, string(types.CollectMethod), &types.ErrorReport{
		Message: suggestion,
	})
}
