package lsp_server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nedpals/bugbuddy-proto/server/daemon"
	daemonClient "github.com/nedpals/bugbuddy-proto/server/daemon/client"
	daemonTypes "github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/nedpals/bugbuddy-proto/server/rpc"
	"github.com/sourcegraph/jsonrpc2"
	lsp "go.lsp.dev/protocol"
)

type LspServer struct {
	conn                   *jsonrpc2.Conn
	daemonClient           *daemonClient.Client
	version                string
	unpublishedDiagnostics []string
	publishChan            chan int
	doneChan               chan int
}

func (s *LspServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	// TODO: add dynamic language registration
	// c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
	// 	Type:    lsp.MessageTypeInfo,
	// 	Message: fmt.Sprintf("[bb-server]: %s", r.Method),
	// })

	switch r.Method {
	case lsp.MethodInitialize:
		c.Reply(ctx, r.ID, lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync:   lsp.TextDocumentSyncKindFull,
				CompletionProvider: nil,
				HoverProvider:      nil,
			},
			ServerInfo: &lsp.ServerInfo{
				Name:    "BugBuddy",
				Version: s.version,
			},
		})
		return
	case lsp.MethodInitialized:
		// c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
		// 	Type:    lsp.MessageTypeInfo,
		// 	Message: "Client is connected",
		// })
		return
	case lsp.MethodShutdown:
		s.daemonClient.Shutdown()
		c.Reply(ctx, r.ID, json.RawMessage("null"))
		return
	case lsp.MethodTextDocumentDidOpen:
		var payload lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		s.daemonClient.ResolveDocument(
			payload.TextDocument.URI.Filename(), // TODO:
			payload.TextDocument.Text,
		)
	case lsp.MethodTextDocumentDidChange:
		var payload lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		// TODO: create a text document store for tracking
		// changes, edit the existing text (if any), and send the
		// newly edited version to the daemon

		// s.daemonClient.UpdateDocument(
		// 	payload.TextDocument.URI.Filename(), // TODO:
		// 	// payload.ContentChanges,
		// )
	case lsp.MethodTextDocumentDidClose:
		var payload lsp.DidCloseTextDocumentParams
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		s.daemonClient.DeleteDocument(
			payload.TextDocument.URI.Filename(),
		)
	case lsp.MethodExit:
		s.doneChan <- 0
		return
	}
}

func Start() error {
	ctx := context.Background()
	doneChan := make(chan int, 1)

	lspServer := &LspServer{
		unpublishedDiagnostics: []string{},
		publishChan:            make(chan int),
		doneChan:               doneChan,
	}

	daemonClient := daemon.NewClient(ctx, daemon.DEFAULT_PORT, daemonTypes.LspClientType, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: fmt.Sprintf("[bugbuddy-client] %s", r.Method),
		})

		if r.Notif && daemonTypes.MethodIs(r.Method, daemonTypes.ReportMethod) {
			lspServer.unpublishedDiagnostics = append(lspServer.unpublishedDiagnostics, "test")
			lspServer.publishChan <- len(lspServer.unpublishedDiagnostics)
		}
	})

	daemonClient.OnReconnect = func() {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: "Daemon not connected. Launching...",
		})
	}

	lspServer.conn = jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(&rpc.CustomStream{
			ReadCloser:  os.Stdin,
			WriteCloser: os.Stdout,
		}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(lspServer),
	)

	lspServer.daemonClient = daemonClient

	if err := daemonClient.Connect(); err != nil {
		if err := daemonClient.EnsureConnection(); err != nil {
			return err
		}
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-exitSignal
		lspServer.doneChan <- 1
	}()

	for {
		select {
		case <-lspServer.publishChan:
			diagnostics := make([]lsp.Diagnostic, len(lspServer.unpublishedDiagnostics))
			for n, errMsg := range lspServer.unpublishedDiagnostics {
				diagnostics[n] = lsp.Diagnostic{
					Severity: lsp.DiagnosticSeverityError,
					Message:  errMsg,
				}
			}

			lspServer.conn.Notify(ctx, lsp.MethodTextDocumentPublishDiagnostics, lsp.PublishDiagnosticsParams{
				Diagnostics: diagnostics,
			})
		case eCode := <-lspServer.doneChan:
			daemonClient.Close()
			lspServer.conn.Close()
			os.Exit(eCode)
		}
	}
}
