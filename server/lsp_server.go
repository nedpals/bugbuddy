package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sourcegraph/jsonrpc2"
	lsp "go.lsp.dev/protocol"
)

const DEFAULT_LSP_PORT = ":3333"

type LspServer struct {
	conn                   *jsonrpc2.Conn
	daemonClient           *DaemonClient
	unpublishedDiagnostics []string
	publishChan            chan int
	doneChan               chan int
}

func (s *LspServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	switch r.Method {
	case "initialize":
		c.Reply(ctx, r.ID, lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.TextDocumentSyncKindNone,
			},
			ServerInfo: &lsp.ServerInfo{
				Name:    "BugBuddy",
				Version: rootCmd.Version,
			},
		})
	case "initialized":
		c.Reply(ctx, r.ID, map[string]string{})
	}
}

func startLspServer(addr string) error {
	lspServer := &LspServer{
		unpublishedDiagnostics: []string{},
		publishChan:            make(chan int),
	}

	lspServer.daemonClient = connectToDaemon(DEFAULT_DAEMON_PORT, CLIENT_TYPE_LSP, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
		if r.Notif && r.Method == "clients/report" {
			lspServer.unpublishedDiagnostics = append(lspServer.unpublishedDiagnostics, "test")
			lspServer.publishChan <- len(lspServer.unpublishedDiagnostics)
		}
	})

	if err := lspServer.daemonClient.EnsureConnection(); err != nil {
		return err
	}

	lspServer.conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(os.Stdin, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(lspServer),
	)

	ctx := context.Background()

	defer func() {
		lspServer.daemonClient.Close()
		lspServer.conn.Close()
	}()

	for {
		select {
		case <-lspServer.publishChan:
			fmt.Println("publishing diagnostics")
			lspServer.conn.Notify(ctx, lsp.MethodTextDocumentPublishDiagnostics, lsp.PublishDiagnosticsParams{
				// TODO:
			})
		case <-lspServer.doneChan:
			return nil
		}
	}
}
