package lsp_server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nedpals/bugbuddy/server/daemon"
	daemonClient "github.com/nedpals/bugbuddy/server/daemon/client"
	daemonTypes "github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/bugbuddy/server/release"
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

type GenerateParticipantIdPayload struct {
	Confirm bool `json:"confirm"`
}

type LspServer struct {
	conn                   *jsonrpc2.Conn
	daemonClient           *daemonClient.Client
	version                string
	unpublishedDiagnostics map[uri.URI][]daemonTypes.ErrorReport
	publishChan            chan int
	doneChan               chan int
	documents              map[uri.URI]*types.Rope
}

func mustDecodePayload[T any](ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) *T {
	if r.Params == nil {
		c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
			Message: "Params field is null",
		})
		return nil
	}

	return decodePayload[T](ctx, c, r)
}

func decodePayload[T any](ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) *T {
	if r.Params == nil {
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
		var dataDirPath string
		customDaemonPort := daemon.DEFAULT_PORT
		payload := decodePayload[lsp.InitializeParams](ctx, c, r)
		if payload != nil {
			if opts, ok := payload.InitializationOptions.(map[string]any); ok {
				if newDataDirPath, ok := opts["data_dir_path"].(string); ok {
					dataDirPath = newDataDirPath
				}

				if newDaemonPort, ok := opts["daemon_port"].(int); ok {
					customDaemonPort = newDaemonPort
				}
			}
		}

		// reinitialize the daemon client to use the custom port
		if customDaemonPort != daemon.DEFAULT_PORT {
			daemon.SetDefaultPort(fmt.Sprintf(":%d", customDaemonPort))
			daemonClient := newDaemonClientForServer(ctx, s)
			daemonClient.SpawnOnMaxReconnect = true
			s.daemonClient = daemonClient
		}

		// connect to the daemon
		if err := s.daemonClient.Connect(); err != nil && err.Error() != "already connected" {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to connect to daemon: %s", err.Error()),
			})
			return
		}

		// change the data dir path on initialize
		if len(dataDirPath) > 0 {
			if err := s.daemonClient.SetDataDirPath(dataDirPath); err != nil {
				c.Reply(ctx, r.ID, lsp.ShowMessageParams{
					Type:    lsp.MessageTypeError,
					Message: fmt.Sprintf("Unable to set data dir: %s", err.Error()),
				})
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
		payload := mustDecodePayload[lsp.DidOpenTextDocumentParams](ctx, c, r)
		if payload == nil {
			return
		}

		// check if file format is excluded
		notFound := true
		ext := filepath.Ext(payload.TextDocument.URI.Filename())
		for _, format := range s.daemonClient.SupportedFileExts() {
			if ext == format {
				notFound = false
				break
			}
		}

		if notFound {
			// ignore the file
			return
		}

		s.documents[payload.TextDocument.URI] = types.NewRope(payload.TextDocument.Text)
		s.daemonClient.ResolveDocument(
			payload.TextDocument.URI.Filename(),
			s.documents[payload.TextDocument.URI].ToString(),
		)

		s.publishChan <- len(s.unpublishedDiagnostics)
	case lsp.MethodTextDocumentDidChange:
		payload := mustDecodePayload[lsp.DidChangeTextDocumentParams](ctx, c, r)
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
		payload := mustDecodePayload[lsp.DidCloseTextDocumentParams](ctx, c, r)
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
	case "$/participantId":
		var participantId string
		if gotParticipantId, err := s.daemonClient.RetrieveParticipantId(); err == nil {
			participantId = gotParticipantId
		} else {
			participantId = "unknown"
		}

		c.Reply(ctx, r.ID, map[string]string{"participant_id": participantId})
		return
	case "$/participantId/generate":
		payload := mustDecodePayload[GenerateParticipantIdPayload](ctx, c, r)
		if payload == nil {
			return
		}

		// check if participant has declined
		if !payload.Confirm {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: "You must confirm to generate a new participant id.",
			})
			return
		}

		newPId, err := s.daemonClient.GenerateParticipantId()
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to refresh participant id: %s", err.Error()),
			})
			return
		}
		c.Reply(ctx, r.ID, map[string]string{"participant_id": newPId})
		return
	case "$/status":
		var participantId string
		if gotParticipantId, err := s.daemonClient.RetrieveParticipantId(); err == nil {
			participantId = gotParticipantId
		} else {
			participantId = "unknown"
		}

		c.Reply(ctx, r.ID, map[string]any{
			"daemon": map[string]any{
				"is_connected": s.daemonClient.IsConnected(),
				"port":         daemon.CurrentPort(),
			},
			"participant_id": participantId,
			"version":        s.version,
		})
		return
	case "$/fetchRunCommand":
		payload := mustDecodePayload[FetchRunCommandPayload](ctx, c, r)
		if payload == nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: "Unable to decode params",
			})
			return
		}

		// get the run command based on language id
		runCommand, err := helpers.GetRunCommand(payload.LanguageId, payload.TextDocument.URI.Filename())
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to run program: %s", err.Error()),
			})
			return
		}

		c.Reply(ctx, r.ID, map[string]string{"command": runCommand})
		return
	case "$/fetchDataDir":
		// call current data dir used by daemon
		dataDir, err := s.daemonClient.GetDataDirPath()
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to fetch data dir: %s", err.Error()),
			})
		}

		c.Reply(ctx, r.ID, map[string]string{"data_dir": dataDir})
		return
	case "$/setDataDir":
		// set the data dir used by daemon
		payload := mustDecodePayload[daemonTypes.SetDataDirRequest](ctx, c, r)
		if payload == nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: "Unable to decode params",
			})
			return
		}

		if err := s.daemonClient.SetDataDirPath(payload.NewPath); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Code:    -32002,
				Message: fmt.Sprintf("Unable to set data dir: %s", err.Error()),
			})
		}

		c.Reply(ctx, r.ID, lsp.ShowMessageParams{
			Type:    lsp.MessageTypeInfo,
			Message: fmt.Sprintf("BugBuddy dir set to %s", payload.NewPath),
		})
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

			uri := uri.File(report.Location.DocumentPath)
			if report.ErrorCode >= 1 {
				lspServer.unpublishedDiagnostics[uri] = []daemonTypes.ErrorReport{
					report,
				}
			} else {
				// clear the diagnostics
				lspServer.unpublishedDiagnostics[uri] = []daemonTypes.ErrorReport{}
			}

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
		unpublishedDiagnostics: map[uri.URI][]daemonTypes.ErrorReport{},
		publishChan:            make(chan int),
		doneChan:               doneChan,
		documents:              map[uri.URI]*types.Rope{},
		version:                release.Version(),
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

			for fileUri, errReports := range lspServer.unpublishedDiagnostics {
				if len(errReports) == 0 {
					// if there are no diagnostics, clear the diagnostics for this file
					diagnosticsMap[fileUri] = []lsp.Diagnostic{}
					continue
				}

				errReport := errReports[0]
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

				// clear the diagnostics for this file
				lspServer.unpublishedDiagnostics[fileUri] = []daemonTypes.ErrorReport{}
			}

			// send the diagnostics to the client
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
