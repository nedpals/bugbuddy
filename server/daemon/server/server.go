package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/rpc"
	"github.com/nedpals/errgoengine"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	engine *errgoengine.ErrgoEngine
	// TODO: add storage for context data
	connectedClients connectedClients
	errors           []string
}

func (d *Server) FS() *SharedFS {
	return d.engine.FS.(*SharedFS)
}

func (d *Server) countLspClients() int {
	count := 0

	for _, cl := range d.connectedClients {
		if cl.clientType == types.LspClientType {
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
		d.connectedClients[info.ProcessId] = connectedClient{
			id:         info.ProcessId,
			clientType: info.ClientType,
			conn:       c,
		}
		c.Reply(ctx, r.ID, 1)
	case types.ShutdownMethod:
		procId := d.getProcessId(r)
		delete(d.connectedClients, procId)
		fmt.Printf("> disconnected: {process_id: %d}\n", procId)
	case types.CollectMethod:
		var payload types.CollectPayload
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		}

		n, err := d.Collect(ctx, payload, c)
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
		} else {
			c.Reply(ctx, r.ID, map[string]any{"n_errors": n})
		}
	case types.PingMethod:
		procId := d.getProcessId(r)
		fmt.Printf("> ping from %d\n", procId)
		c.Reply(ctx, r.ID, "pong!")
	case types.ResolveDocumentMethod:
		var payloadStr types.DocumentPayload
		if err := json.Unmarshal(*r.Params, &payloadStr); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		d.FS().WriteFile(payloadStr.Filepath, []byte(payloadStr.Content))
		fmt.Printf("> resolved document: %s (len: %d)\n", payloadStr.Filepath, len(payloadStr.Content))
	case types.UpdateDocumentMethod:
		var payloadStr types.DocumentPayload
		if err := json.Unmarshal(*r.Params, &payloadStr); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		// IDEA: create a dependency tree wherein errors will be removed
		// once the file is updated
		d.FS().WriteFile(payloadStr.Filepath, []byte(payloadStr.Content))
		fmt.Printf("> updated document: %s (len: %d)\n", payloadStr.Filepath, len(payloadStr.Content))
	case types.DeleteDocumentMethod:
		var payload types.DocumentIdentifier
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		// TODO: use dependency tree
		d.FS().Remove(payload.Filepath)
		fmt.Printf("> removed document: %s\n", payload.Filepath)
	}
}

func (s *Server) Collect(ctx context.Context, payload types.CollectPayload, c *jsonrpc2.Conn) (int, error) {
	// fmt.Println(err)
	// TODO: idk what to do with this lol
	// s.errors = append(s.errors, err)

	// TODO: process error first before notify
	fmt.Printf("> report new errors to %d clients\n", s.countLspClients())

	s.engine.Analyze(
		payload.WorkingDir,
		payload.Error,
	)

	s.connectedClients.Notify(ctx, types.ReportMethod, &types.ErrorReport{
		Message: payload.Error,
	}, types.LspClientType)

	return 1, nil
}

func Start(addr string) error {
	fmt.Println("> daemon started on " + addr)
	return rpc.StartServer(addr, jsonrpc2.VarintObjectCodec{}, &Server{
		engine: &errgoengine.ErrgoEngine{
			ErrorTemplates: errgoengine.ErrorTemplates{},
			FS:             NewSharedFS(),
		},
		connectedClients: connectedClients{},
		errors:           []string{},
	})
}
