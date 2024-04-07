package server_test

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/nedpals/bugbuddy/server/daemon/client"
	"github.com/nedpals/bugbuddy/server/daemon/server"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/errgoengine"
	"github.com/nedpals/errgoengine/languages"
	"github.com/sourcegraph/jsonrpc2"
)

const defaultAddr = ":3434"

func Setup() (*jsonrpc2.Conn, *server.Server, *client.Client) {
	server := server.NewServer()
	languages.SupportedLanguages = append(languages.SupportedLanguages, errgoengine.TestLanguage)

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

	client := client.NewClient(context.Background(), defaultAddr, types.MonitorClientType)
	client.SetConn(clientConn)

	return conn, server, client
}

func TestHandshake(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// check if the server has a client
	gotClientId, gotClientType := srv.Clients().Get(clientId)
	if gotClientId != clientId {
		t.Fatalf("expected client id %d, got %d", clientId, gotClientId)
	}

	if gotClientType != types.MonitorClientType {
		t.Fatalf("expected client type %v, got %v", types.MonitorClientType, gotClientType)
	}
}

func TestShutdown(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// client shutdown
	err := client.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	client.Close()

	// check if client is still connected
	if client.IsConnected() {
		t.Fatalf("expected client to be disconnected")
	}

	if id, cType := srv.Clients().Get(clientId); id != -1 && cType != types.UnknownClientType {
		t.Fatalf("expected client to be disconnected from the server")
	}
}

func TestServerShutdown(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// server shutdown
	srv.Clients().Disconnect()

	// check if the server has no clients
	if len(srv.Clients()) != 0 {
		t.Fatalf("expected all clients to be disconnected")
	}
}

func TestResolveDocument(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("hello.py", "print(a)")
	if err != nil {
		t.Fatal(err)
	}

	// wait for the server to process the document
	time.Sleep(200 * time.Millisecond)

	// check if the server has the document
	file, err := srv.FS().Open("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if stat.Name() != "hello.py" {
		t.Fatalf("expected file name hello.py, got %s", stat.Name())
	}

	if stat.Size() == 0 {
		t.Fatalf("expected file size > 0, got %d", stat.Size())
	}

	// check file contents
	contents := make([]byte, stat.Size())
	_, err = file.Read(contents)

	if err != nil {
		t.Fatal(err)
	}

	if string(contents) != "print(a)" {
		t.Fatalf("expected file contents print(a), got %s", string(contents))
	}
}

func TestResolveDocument_EmptyFilepath(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("", "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDeleteDocument(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("hello.py", "print(a)")
	if err != nil {
		t.Fatal(err)
	}

	if srv.GetFileUseIdx("hello.py", clientId) == -1 {
		t.Fatalf("expected client to be added as user of the file")
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// delete the document
	err = client.DeleteDocument("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	// check if the client is removed as user of the file
	if srv.GetFileUseIdx("hello.py", clientId) != -1 {
		t.Fatalf("expected client to be removed as user of the file")
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// check if the server has the document
	_, err = srv.FS().Open("hello.py")
	if err == nil {
		t.Fatalf("expected file not found error, got nil")
	}
}

func TestDeleteDocument_EmptyFilepath(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// delete the document
	err := client.DeleteDocument("")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDeleteDocument_AlreadyDeleted(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// create the document
	err := client.ResolveDocument("hello.py", "print(a)")
	if err != nil {
		t.Fatal(err)
	}

	// delete the document
	err = client.DeleteDocument("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// delete the document again
	err = client.DeleteDocument("hello.py")
	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "File does not exist" {
			t.Fatalf("expected File does not exist error, got %s", jErr.Message)
		}
	} else {
		t.Fatalf("expected jsonrpc2.Error, got %T", err)
	}
}

func TestUpdateDocument(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("hello.py", "print(a)")
	if err != nil {
		t.Fatal(err)
	}

	if srv.GetFileUseIdx("hello.py", clientId) == -1 {
		t.Fatalf("expected client to be added as user of the file")
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// update the document
	err = client.UpdateDocument("hello.py", "print(b)")
	if err != nil {
		t.Fatal(err)
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// check file contents
	file, err := srv.FS().Open("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	contents := make([]byte, stat.Size())
	_, err = file.Read(contents)

	if err != nil {
		t.Fatal(err)
	}

	if string(contents) != "print(b)" {
		t.Fatalf("expected file contents print(b), got %s", string(contents))
	}
}

func TestUpdateDocument_EmptyFilepath(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// update the document
	err := client.UpdateDocument("", "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUpdateDocument_Nonexisting(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// update the document
	err := client.UpdateDocument("hello.py", "print(a)")
	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "File does not exist" {
			t.Fatalf("expected File does not exist error, got %s", jErr.Message)
		}
	} else {
		t.Fatalf("expected jsonrpc2.Error, got %T", err)
	}
}

func TestRetrieveDocument(t *testing.T) {
	clientId := 1
	conn, srv, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// add first document
	expected := "print(a)"
	err := client.ResolveDocument("hello.py", expected)
	if err != nil {
		t.Fatal(err)
	}

	if srv.GetFileUseIdx("hello.py", clientId) == -1 {
		t.Fatalf("expected client to be added as user of the file")
	}

	// retrieve the document
	resp, err := client.RetrieveDocument("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	if resp != expected {
		t.Fatalf("expected non-empty content, got %s", resp)
	}
}

func TestCollect(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("Hello.java", `public class Hello {
	public static void main(String[] args) {
		String a = null;
		System.out.println(a);
	}
}`)
	if err != nil {
		t.Fatal(err)
	}

	// collect the error
	resp, err := client.Collect(1, "java Hello", ".", `Exception in thread "main" java.lang.NullPointerException
	at Hello.main(Hello.java:4)`)

	if err != nil {
		t.Fatal(err)
	}

	if resp.Recognized != 1 {
		t.Fatalf("expected 1 error, got %d", resp.Recognized)
	}

	if resp.Processed != 1 {
		t.Fatalf("expected 1 processed, got %d", resp.Processed)
	}
}

func TestCollect_ShouldError(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// load the document
	err := client.ResolveDocument("Hello.test", `i'm a dummy program`)
	if err != nil {
		t.Fatal(err)
	}

	// collect the error
	resp, err := client.Collect(1, "cat Hello.test", ".", `im an error!`)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if resp.Recognized != 0 {
		t.Fatalf("expected 0 received, got %d", resp.Recognized)
	}

	if resp.Processed != 0 {
		t.Fatalf("expected 0 processed, got %d", resp.Processed)
	}
}

func TestGenerateParticipantID(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// generate participant id
	participantId, err := client.GenerateParticipantId()
	if err != nil {
		t.Fatal(err)
	}

	if len(participantId) == 0 {
		t.Fatalf("expected participant id to be generated")
	}
}

func TestGenerateParticipantIDReset(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// get existing participant id
	participantId, err := client.RetrieveParticipantId()
	if err != nil {
		t.Fatal(err)
	}

	if len(participantId) == 0 {
		t.Fatalf("expected participant id to be generated")
	}

	// generate participant id again
	newParticipantId, err := client.GenerateParticipantId()
	if err != nil {
		t.Fatal(err)
	}

	if len(participantId) == 0 {
		t.Fatalf("expected participant id to be generated")
	}

	if participantId == newParticipantId {
		t.Fatalf("expected new participant id to be different. got %s", newParticipantId)
	}
}

func TestResetLogger(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// reset logger
	err := client.ResetLogger()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCall_NoProcessId(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// call without process id, negative process ids are
	// not included when making a request
	client.SetId(-3)
	err := client.Call("test", "test", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "Process ID not found" {
			t.Fatalf("expected error message Process ID not found, got %s", jErr.Message)
		}
	} else {
		t.Fatalf("expected jsonrpc2.Error, got %T", err)
	}
}

func TestCall_InvalidProcessId(t *testing.T) {
	clientId := 1
	conn, _, client := Setup()
	defer conn.Close()

	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// call
	client.SetId(111)
	err := client.Call("test", "test", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if jErr, ok := err.(*jsonrpc2.Error); ok {
		if jErr.Message != "Process not connected yet." {
			t.Fatalf("expected error message Process not connected yet., got %s", jErr.Message)
		}
	} else {
		t.Fatalf("expected jsonrpc2.Error, got %T", err)
	}
}
