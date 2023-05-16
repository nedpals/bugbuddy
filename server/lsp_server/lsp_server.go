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

const DEFAULT_PORT = ":3333"

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
	c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
		Type:    lsp.MessageTypeInfo,
		Message: fmt.Sprintf("[bb-server]: %s", r.Method),
	})

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
		// c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
		// 	Type:    lsp.MessageTypeInfo,
		// 	Message: fmt.Sprintf("[bb-server]: %s", r.Method),
		// })
	case lsp.MethodExit:
		s.doneChan <- 0
		return
	}
}

func Start(addr string) error {
	doneChan := make(chan int, 1)

	lspServer := &LspServer{
		unpublishedDiagnostics: []string{},
		publishChan:            make(chan int),
		doneChan:               doneChan,
	}

	lspServer.conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(&rpc.CustomStream{
			ReadCloser:  os.Stdin,
			WriteCloser: os.Stdout,
		}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(lspServer),
	)

	daemonClient := daemon.NewClient(daemon.DEFAULT_PORT, daemonTypes.LspClientType, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
		fmt.Println(r.Method)

		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: fmt.Sprintf("[bugbuddy-client] %s", r.Method),
		})

		if r.Notif && daemonTypes.MethodIs(r.Method, daemonTypes.ReportMethod) {
			lspServer.unpublishedDiagnostics = append(lspServer.unpublishedDiagnostics, "test")
			lspServer.publishChan <- len(lspServer.unpublishedDiagnostics)
		}
	})

	ctx := context.Background()
	daemonClient.OnReconnect = func() {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: "Daemon not connected. Launching...",
		})
	}

	if err := daemonClient.Connect(); err != nil {
		if err := daemonClient.EnsureConnection(); err != nil {
			return err
		}
	}

	lspServer.daemonClient = daemonClient
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
