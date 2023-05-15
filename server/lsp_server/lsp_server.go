package lsp_server

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/nedpals/bugbuddy-proto/server/daemon"
	daemonClient "github.com/nedpals/bugbuddy-proto/server/daemon/client"
	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
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
	c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
		Type:    lsp.MessageTypeInfo,
		Message: r.Method,
	})

	switch r.Method {
	case lsp.MethodInitialize:
		c.Reply(ctx, r.ID, lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.TextDocumentSyncKindIncremental,
			},
			ServerInfo: &lsp.ServerInfo{
				Name:    "BugBuddy",
				Version: s.version,
			},
		})
		return
	case lsp.MethodInitialized:
		c.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: "Client is connected",
		})
	case lsp.MethodExit:
		s.doneChan <- 0
		return
	}
}

type connection struct {
	io.ReadCloser
	io.WriteCloser
}

func (conn *connection) Read(p []byte) (n int, err error) {
	return conn.ReadCloser.Read(p)
}

func (conn *connection) Write(p []byte) (n int, err error) {
	return conn.WriteCloser.Write(p)
}

func (conn *connection) Close() error {
	if err := conn.ReadCloser.Close(); err != nil {
		return err
	} else if err := conn.WriteCloser.Close(); err != nil {
		return err
	}
	return nil
}

func Start(addr string) error {
	lspServer := &LspServer{
		unpublishedDiagnostics: []string{},
		publishChan:            make(chan int),
		doneChan:               make(chan int, 1),
	}

	lspServer.conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(&connection{
			ReadCloser:  os.Stdin,
			WriteCloser: os.Stdout,
		}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(lspServer),
	)

	daemonClient := daemon.Connect(daemon.DEFAULT_PORT, types.LspClientType, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: r.Method,
		})

		if r.Notif && r.Method == "clients/report" {
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

	if err := daemonClient.EnsureConnection(); err != nil {
		return err
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
			lspServer.daemonClient.Close()
			lspServer.conn.Close()
			os.Exit(eCode)
		}
	}
}
