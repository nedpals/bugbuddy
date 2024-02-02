package lsp_server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/nedpals/bugbuddy/server/daemon"
	daemonClient "github.com/nedpals/bugbuddy/server/daemon/client"
	daemonTypes "github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/bugbuddy/server/rpc"
	"github.com/nedpals/bugbuddy/server/types"
	"github.com/sourcegraph/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// type ViewErrorPayload struct {
// 	Id int `json:"id"`
// }

type FetchRunCommandPayload struct {
	LanguageId   string                     `json:"languageId"`
	TextDocument lsp.TextDocumentIdentifier `json:"textDocument"`
}

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
	if r.Params == nil {
		c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
			Message: "Params field is null",
		})
		return nil
	}

	var payload *T
	if err := json.Unmarshal(*r.Params, &payload); err != nil {
		c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
			Message: "Unable to decode params of method " + r.Method + ": " + err.Error(),
		})
		return nil
	}
	return payload
}

func (s *LspServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	// TODO: add dynamic language registration

	switch r.Method {
	case lsp.MethodInitialize:
		if !s.daemonClient.IsConnected() {
			if err := s.daemonClient.Connect(); err != nil {
				c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
					Code:    -32002,
					Message: fmt.Sprintf("Unable to connect to daemon: %s", err.Error()),
				})
				return
			}
		}

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
	case "$/fetchRunCommand":
		payload := decodePayload[FetchRunCommandPayload](ctx, c, r)
		if payload == nil {
			return
		}

		// get the run command based on language id
		runCommand, err := helpers.GetRunCommand(payload.LanguageId, payload.TextDocument.URI.Filename())
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to run program: %s", err.Error()),
			})
		}

		c.Reply(ctx, r.ID, map[string]string{"command": runCommand})
		return
	case lsp.MethodExit:
		s.doneChan <- 0
		return
	}
}

func newDaemonClientForServer(ctx context.Context, lspServer *LspServer) *daemon.Client {
	daemonClient := daemon.NewClient(ctx, daemon.CurrentPort(), daemonTypes.LspClientType, func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
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

	daemonClient.OnReconnect = func(retries int, _ error) bool {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: "Reconnecting...",
		})
		return retries <= 5
	}

	daemonClient.OnSpawnDaemon = func() {
		lspServer.conn.Notify(ctx, lsp.MethodWindowShowMessage, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: "Daemon not connected. Launching...",
		})
	}

	return daemonClient
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

	lspServer.conn = jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(&rpc.CustomStream{
			ReadCloser:  os.Stdin,
			WriteCloser: os.Stdout,
		}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(lspServer),
	)

	daemonClient := newDaemonClientForServer(ctx, lspServer)
	daemonClient.SpawnOnMaxReconnect = true

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
			diagnosticsMap := map[uri.URI][]lsp.Diagnostic{}

			for _, errReport := range lspServer.unpublishedDiagnostics {
				fileUri := uri.File(errReport.Location.DocumentPath)
				tempExpFilepath := getTempFilePath(fileUri.Filename())
				openErrorRawUri := fmt.Sprintf("vscode://nedpals.bugbuddy/openError?file=%s", url.QueryEscape(tempExpFilepath))
				openErrorUri := uri.URI(openErrorRawUri)

				diagnosticsMap[fileUri] = append(diagnosticsMap[fileUri], lsp.Diagnostic{
					Severity: lsp.DiagnosticSeverityError,
					Message:  fmt.Sprintf("%s\n\nClick the error code for more details.", errReport.Message),
					Code:     fmt.Sprintf("%s/%s", errReport.Language, errReport.Template),
					CodeDescription: &lsp.CodeDescription{
						Href: openErrorUri,
					},
					Source: "BugBuddy",
					Range: lsp.Range{
						Start: lsp.Position{
							Line:      uint32(errReport.Location.StartPos.Line),
							Character: uint32(errReport.Location.StartPos.Column),
						},
						End: lsp.Position{
							Line:      uint32(errReport.Location.EndPos.Line),
							Character: uint32(errReport.Location.EndPos.Column),
						},
					},
				})

				// save the output into a temporary file
				if file, err := getTempFileForFile(fileUri.Filename()); err == nil {
					file.WriteString(errReport.FullMessage)
					file.Close()
				}
			}

			// delete all unpublished diagnostics
			lspServer.unpublishedDiagnostics = []daemonTypes.ErrorReport{}

			for uri, diagnostics := range diagnosticsMap {
				lspServer.conn.Notify(ctx, lsp.MethodTextDocumentPublishDiagnostics, lsp.PublishDiagnosticsParams{
					URI: uri,
					// TODO: version
					Diagnostics: diagnostics,
				})
			}
		case eCode := <-lspServer.doneChan:
			removeAllTempFiles()
			daemonClient.Close()
			lspServer.conn.Close()
			os.Exit(eCode)
		}
	}
}
