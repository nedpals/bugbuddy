package daemon_test

import (
	"context"
	"testing"

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

	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// check if client is connected
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}

	// check if the server has a client
	gotClientId, gotClientType := server.GetClientInfo(srv, clientId)
	if gotClientId != clientId {
		t.Fatalf("expected client id %d, got %d", clientId, gotClientId)
	}

	if gotClientType != types.MonitorClientType {
		t.Fatalf("expected client type %v, got %v", types.MonitorClientType, gotClientType)
	}

	defer client.Close()
}
