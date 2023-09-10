package lsp_server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nedpals/bugbuddy/server/daemon"
	daemonClient "github.com/nedpals/bugbuddy/server/daemon/client"
	daemonTypes "github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
	"github.com/nedpals/bugbuddy/server/types"
	"github.com/sourcegraph/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type LspServer struct {
	conn                   *jsonrpc2.Conn
	daemonClient           *daemonClient.Client
	version                string
	unpublishedDiagnostics []daemonTypes.ErrorReport
	publishChan            chan int
	doneChan               chan int
	documents              map[uri.URI]*types.Rope
}

func decodePayload[T any](ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) *T {
	var payload *T
	if err := json.Unmarshal(*r.Params, &payload); err != nil {
		c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
			Message: "Unable to decode params of method " + r.Method,
		})
		return nil
	}
	return payload
}

func (s *LspServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	// TODO: add dynamic language registration

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
		payload := decodePayload[lsp.DidOpenTextDocumentParams](ctx, c, r)
		if payload == nil {
			return
		}

		s.documents[payload.TextDocument.URI] = types.NewRope(payload.TextDocument.Text)
		s.daemonClient.ResolveDocument(
			payload.TextDocument.URI.Filename(),
			s.documents[payload.TextDocument.URI].ToString(),
		)

		s.publishChan <- len(s.unpublishedDiagnostics)
	case lsp.MethodTextDocumentDidChange:
		payload := decodePayload[lsp.DidChangeTextDocumentParams](ctx, c, r)
		if payload == nil {
			return
		}

		text := s.documents[payload.TextDocument.URI]

		// edit the existing text (if any), and send the newly edited version to the daemon
		for _, change := range payload.ContentChanges {
			startOffset := text.OffsetFromPosition(change.Range.Start)

			if len(change.Text) == 0 {
				endOffset := text.OffsetFromPosition(change.Range.End)
				text.Delete(startOffset, endOffset-startOffset)
			} else {
				text.Insert(startOffset, change.Text)
			}
		}

		s.daemonClient.UpdateDocument(
			payload.TextDocument.URI.Filename(), // TODO:
			text.ToString(),
		)
	case lsp.MethodTextDocumentDidClose:
		payload := decodePayload[lsp.DidCloseTextDocumentParams](ctx, c, r)
		if payload == nil {
			return
		}

		delete(s.documents, payload.TextDocument.URI)
		s.daemonClient.DeleteDocument(
			payload.TextDocument.URI.Filename(),
		)

		s.conn.Notify(ctx, lsp.MethodTextDocumentPublishDiagnostics, lsp.PublishDiagnosticsParams{
			URI: payload.TextDocument.URI,
			// TODO: version
			Diagnostics: []lsp.Diagnostic{},
		})
	// case lsp.MethodTextDocumentHover:
	// 	payload := decodePayload[lsp.HoverParams](ctx, c, r)
	// 	if payload == nil {
	// 		return
	// 	}

	// 	err := s.daemonClient.Call(
	// 		daemonTypes.NearestNodeMethod,
	// 		daemonTypes.NearestNodePayload{
	// 			Line: int(payload.Position.Line),
	// 			Column: int(payload.Position.Character),
	// 			DocumentIdentifier: daemonTypes.DocumentIdentifier{
	// 				DocumentPath: payload.TextDocument.URI.Filename()
	// 			},
	// 		},
	// 	)
	// 	if err != nil {

	// 	}
	case lsp.MethodExit:
		s.doneChan <- 0
		return
	}
}

func Start() error {
	ctx := context.Background()
	doneChan := make(chan int, 1)

	lspServer := &LspServer{
		unpublishedDiagnostics: []daemonTypes.ErrorReport{},
		publishChan:            make(chan int),
		doneChan:               doneChan,
		documents:              map[uri.URI]*types.Rope{},
		version:                "1.0",
	}

	daemonClient := daemon.NewClient(ctx, daemon.DEFAULT_PORT, daemonTypes.LspClientType, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
		if r.Notif && daemonTypes.MethodIs(r.Method, daemonTypes.ReportMethod) {
			var report daemonTypes.ErrorReport
			if err := json.Unmarshal(*r.Params, &report); err != nil {
				lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
					Type:    lsp.MessageTypeError,
					Message: fmt.Sprintf("unable to report error: %s", err.Error()),
				})
				return
			}

			lspServer.unpublishedDiagnostics = append(lspServer.unpublishedDiagnostics, report)
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
			diagnosticsMap := map[uri.URI][]lsp.Diagnostic{}
			for _, errReport := range lspServer.unpublishedDiagnostics {
				uri := uri.File(errReport.Location.DocumentPath)
				diagnosticsMap[uri] = append(diagnosticsMap[uri], lsp.Diagnostic{
					Severity: lsp.DiagnosticSeverityError,
					Message:  errReport.Message,
					Code:     fmt.Sprintf("%s/%s", errReport.Language, errReport.Template),
					Source:   "BugBuddy",
					Range: lsp.Range{
						Start: lsp.Position{
							Line:      uint32(errReport.Location.Line),
							Character: uint32(errReport.Location.Column),
						},
						End: lsp.Position{
							Line:      uint32(errReport.Location.Line),
							Character: uint32(errReport.Location.Column),
						},
					},
				})
			}

			// lspServer.unpublishedDiagnostics = nil

			for uri, diagnostics := range diagnosticsMap {
				lspServer.conn.Notify(ctx, lsp.MethodTextDocumentPublishDiagnostics, lsp.PublishDiagnosticsParams{
					URI: uri,
					// TODO: version
					Diagnostics: diagnostics,
				})
			}
		case eCode := <-lspServer.doneChan:
			daemonClient.Close()
			lspServer.conn.Close()
			os.Exit(eCode)
		}
	}
}
