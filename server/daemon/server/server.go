package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/rpc"
	"github.com/nedpals/errgoengine"
	"github.com/nedpals/errgoengine/error_templates"
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
			fmt.Printf("> collect error: %s\n", err.Error())
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
	template, data, err := s.engine.Analyze(payload.WorkingDir, payload.Error)
	if err != nil {
		return 0, err
	}

	output := s.engine.Translate(template, data)

	fmt.Printf("> report new errors to %d clients\n", s.countLspClients())
	s.connectedClients.Notify(ctx, types.ReportMethod, &types.ErrorReport{
		Message:  output,
		Template: template.Name,
		Language: template.Language.Name,
		Location: data.MainError.Nearest.Location(),
	}, types.LspClientType)

	return 1, nil
}

func Start(addr string) error {
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	errChan := make(chan error, 1)
	disconnChan := make(chan int, 1)
	exitSignal := make(chan os.Signal, 1)

	errorTemplates := errgoengine.ErrorTemplates{}
	error_templates.LoadErrorTemplates(&errorTemplates)
	server := &Server{
		engine: &errgoengine.ErrgoEngine{
			ErrorTemplates: errorTemplates,
			FS:             NewSharedFS(),
		},
		connectedClients: connectedClients{},
		errors:           []string{},
	}

	go func() {
		fmt.Println("> daemon started on " + addr)
		errChan <- rpc.StartServer(
			addr,
			jsonrpc2.VarintObjectCodec{},
			server,
		)
	}()

	signal.Notify(exitSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSignal
		disconnChan <- 1
	}()

	for {
		select {
		case err := <-errChan:
			return err
		case <-time.After(15 * time.Second):
			// Disconnect only if CTRL+C is pressed or is launched
			// as a background terminal
			if !isTerminal && len(server.connectedClients) == 0 {
				disconnChan <- 1
			}
		case <-disconnChan:
			server.connectedClients.Disconnect()
			return nil
		}
	}
}
