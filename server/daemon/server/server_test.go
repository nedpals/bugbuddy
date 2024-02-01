package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/nedpals/bugbuddy/server/daemon/client"
	"github.com/nedpals/bugbuddy/server/daemon/server"
	"github.com/nedpals/bugbuddy/server/daemon/types"
)

const defaultAddr = ":3434"

func StartServer() *server.Server {
	server := server.NewServer()
	go func() {
		server.Start(defaultAddr)
	}()
	return server
}

func TestHandshake(t *testing.T) {
	clientId := 1
	srv := StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
	client.SetId(clientId)
	defer client.Close()

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

	// check if client is still connected
	if client.IsConnected() {
		t.Fatalf("expected client to be disconnected")
	}
}

func TestResolveDocument(t *testing.T) {
	clientId := 1
	srv := StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	time.Sleep(100 * time.Millisecond)

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

func TestResolveDocument_InvalidParams(t *testing.T) {
	clientId := 1
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	err := client.Notify(types.ResolveDocumentMethod, 1)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDeleteDocument(t *testing.T) {
	clientId := 1
	srv := StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	time.Sleep(100 * time.Millisecond)

	// delete the document
	err = client.DeleteDocument("hello.py")
	if err != nil {
		t.Fatal(err)
	}

	// wait for the server to process the document
	time.Sleep(100 * time.Millisecond)

	// check if the server has the document
	_, err = srv.FS().Open("hello.py")
	if err == nil {
		t.Fatalf("expected file not found error, got nil")
	}
}

func TestUpdateDocument(t *testing.T) {
	clientId := 1
	srv := StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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

func TestCollect(t *testing.T) {
	clientId := 1
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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

	// collect the error
	received, processed, err := client.Collect(1, "python3 hello.py", ".", `Traceback (most recent call last):
	  File "./test_programs/dangling.py", line 1, in <module>
		print(name)
			  ^^^^
	NameError: name 'name' is not defined`)

	// if err != nil {
	// 	t.Fatal(err)
	// }

	if received != 1 {
		t.Fatalf("expected 1 error, got %d", received)
	}

	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}
}

func TestGenerateParticipantID(t *testing.T) {
	clientId := 1
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
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
	_ = StartServer()

	client := client.NewClient(context.TODO(), defaultAddr, types.MonitorClientType)
	client.SetId(clientId)
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// call without process id
	client.SetId(0)
	err := client.Call("test", "test", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
