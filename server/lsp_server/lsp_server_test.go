package lsp_server

import (
	"context"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/nedpals/bugbuddy/server/daemon/server"
	daemonTypes "github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
	"github.com/nedpals/bugbuddy/server/types"
	"github.com/sourcegraph/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func daemonConnSetup() (*jsonrpc2.Conn, net.Conn) {
	server := server.NewServer()
	server.ServerLog = log.New(io.Discard, "", log.LstdFlags)

	serverConn, clientConn := net.Pipe()

	conn := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(
			serverConn,
			&jsonrpc2.VarintObjectCodec{},
		),
		server,
	)

	return conn, clientConn
}

func Setup() (func(), *LspServer, *rpc.Client) {
	daemonServerConn, daemonClientConn := daemonConnSetup()
	serverConn, clientConn := net.Pipe()
	// Create a mock LspServer
	lspServer := &LspServer{
		unpublishedDiagnostics: []daemonTypes.ErrorReport{},
		publishChan:            make(chan int, 0),
		doneChan:               make(chan int, 0),
		documents:              map[uri.URI]*types.Rope{},
		version:                "1.0",
	}

	// Connect piped serverConn to a jsonrpc2.Conn
	lspServer.conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(serverConn, jsonrpc2.VSCodeObjectCodec{}),
		lspServer,
	)

	// Connect daemon client
	daemonClient := newDaemonClientForServer(context.Background(), lspServer)
	daemonClient.SetConn(daemonClientConn)
	lspServer.daemonClient = daemonClient

	// Create a client for lsp
	client := &rpc.Client{}
	client.Conn = jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(clientConn, jsonrpc2.VSCodeObjectCodec{}),
		client,
	)

	return func() {
		daemonServerConn.Close()
		lspServer.conn.Close()
	}, lspServer, client
}

func initialize(lspServer *LspServer, client *rpc.Client) (lsp.InitializeResult, error) {
	var result lsp.InitializeResult
	err := client.Call(lsp.MethodInitialize, nil, &result)
	return result, err
}

func TestInitialize(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	result, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	exp := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync:   lsp.TextDocumentSyncKindFull,
			CompletionProvider: nil,
			HoverProvider:      nil,
		},
		ServerInfo: &lsp.ServerInfo{
			Name:    "BugBuddy",
			Version: srv.version,
		},
	}

	if result.Capabilities.TextDocumentSync == nil {
		t.Error("Expected TextDocumentSync to be non-nil")
	}

	if tdSync := lsp.TextDocumentSyncKind(result.Capabilities.TextDocumentSync.(float64)); tdSync != exp.Capabilities.TextDocumentSync {
		t.Errorf("Expected %v, got %v", exp.Capabilities.TextDocumentSync, tdSync)
	}

	if exp.ServerInfo.Name != result.ServerInfo.Name {
		t.Errorf("Expected %v, got %v", exp.ServerInfo.Name, result.ServerInfo.Name)
	}

	if exp.ServerInfo.Version != result.ServerInfo.Version {
		t.Errorf("Expected %v, got %v", exp.ServerInfo.Version, result.ServerInfo.Version)
	}
}

func TestInitialized(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	err = client.Notify(lsp.MethodInitialized, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestShutdown(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	var result interface{}
	err = client.Call(lsp.MethodShutdown, nil, &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}

	if srv.daemonClient.IsConnected() {
		t.Error("Expected daemon client to be disconnected")
	}
}

func TestExit(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	err = client.Notify(lsp.MethodExit, nil)
	if err != nil {
		t.Fatal(err)
	}

	result := <-srv.doneChan
	if result != 0 {
		t.Errorf("Expected 0, got %v", result)
	}
}

func TestMethodTextDocumentDidOpen(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	err = client.Notify(lsp.MethodTextDocumentDidOpen, lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the document to be opened
	time.Sleep(100 * time.Millisecond)

	<-srv.publishChan

	if _, ok := srv.documents[uri.URI("file:///test.go")]; !ok {
		t.Error("Expected document to be opened")
	}
}

func TestMethodTextDocumentDidOpen_NoPayload(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	client.Notify(lsp.MethodTextDocumentDidOpen, nil)
}

func TestMethodTextDocumentDidChange(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	// open document first
	err = client.Notify(lsp.MethodTextDocumentDidOpen, lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the document to be opened
	time.Sleep(100 * time.Millisecond)

	<-srv.publishChan

	if _, ok := srv.documents[uri.URI("file:///test.go")]; !ok {
		t.Error("Expected document to be opened")
	}

	// change document
	err = client.Notify(lsp.MethodTextDocumentDidChange, lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: uri.URI("file:///test.go"),
			},
			Version: 1,
		},
		ContentChanges: []lsp.TextDocumentContentChangeEvent{
			{
				Text: "",
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      0,
						Character: 0,
					},
					End: lsp.Position{
						Line:      0,
						Character: 12,
					},
				},
				RangeLength: 12,
			},
			{
				Text: "package main2",
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      0,
						Character: 0,
					},
					End: lsp.Position{
						Line:      0,
						Character: 0,
					},
				},
				RangeLength: 0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the document to be changed
	time.Sleep(100 * time.Millisecond)

	if _, ok := srv.documents[uri.URI("file:///test.go")]; !ok {
		t.Error("Expected document to be changed")
	}

	if srv.documents[uri.URI("file:///test.go")].ToString() != "package main2" {
		t.Errorf("Expected %v, got %v", "package main", srv.documents[uri.URI("file:///test.go")].ToString())
	}
}

func TestMethodTextDocumentDidChange_NoPayload(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	client.Notify(lsp.MethodTextDocumentDidChange, nil)
}

func TestMethodTextDocumentDidClose(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	// open document first
	err = client.Notify(lsp.MethodTextDocumentDidOpen, lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the document to be opened
	time.Sleep(100 * time.Millisecond)

	<-srv.publishChan

	if _, ok := srv.documents[uri.URI("file:///test.go")]; !ok {
		t.Error("Expected document to be opened")
	}

	// close document
	err = client.Notify(lsp.MethodTextDocumentDidClose, lsp.DidCloseTextDocumentParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: uri.URI("file:///test.go"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the document to be closed
	time.Sleep(100 * time.Millisecond)

	if _, ok := srv.documents[uri.URI("file:///test.go")]; ok {
		t.Error("Expected document to be closed")
	}
}

func TestMethodTextDocumentDidClose_NoPayload(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	client.Notify(lsp.MethodTextDocumentDidClose, nil)
}

func TestFetchRunCommand(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	err = client.Call("$/fetchRunCommand", FetchRunCommandPayload{
		LanguageId: "go",
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.go",
		},
	}, &result)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(result["command"], " -- go run /test.go") {
		t.Errorf("Expected %v, got %v", "go run test.go", result["command"])
	}
}

func TestFetchRunCommand_NoPayload(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	err = client.Call("$/fetchRunCommand", nil, &result)
	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "Params field is null" {
			t.Errorf("Expected %v, got %v", "Params field is null", jErr.Message)
		}
	} else {
		t.Fatal(err)
	}

	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestFetchRunCommand_InvalidLanguageId(t *testing.T) {
	close, srv, client := Setup()
	defer close()

	_, err := initialize(srv, client)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	err = client.Call("$/fetchRunCommand", FetchRunCommandPayload{
		LanguageId: "invalid",
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///test.go",
		},
	}, &result)
	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "Unable to run program: no run command for language id invalid" {
			t.Errorf("Expected %v, got %v", "Unable to run program: no run command for language id invalid", jErr.Message)
		}
	} else {
		t.Fatal(err)
	}

	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}
