package daemon

import (
	"context"
	"fmt"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon/client"
	"github.com/nedpals/bugbuddy/server/daemon/server"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
)

type Client = client.Client

const DEFAULT_PORT = 3434

var currentPort = fmt.Sprintf("%d", DEFAULT_PORT)

func init() {
	if port := os.Getenv("BUGBUDDY_DAEMON_PORT"); len(port) > 0 {
		SetDefaultPort(port)
	}
}

func NewClient(ctx context.Context, addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *client.Client {
	return client.NewClient(ctx, addr, clientType, handlerFunc...)
}

func Connect(addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *client.Client {
	return client.Connect(addr, clientType, handlerFunc...)
}

func Serve(addr string) error {
	return server.Start(addr)
}

func Execute(clientType types.ClientType, execFn func(client *Client) error) error {
	client := NewClient(context.Background(), CurrentPort(), clientType)
	client.OnReconnect = func(retries int, err error) bool {
		fmt.Println("reconnecting...")
		return retries <= 5
	}
	client.SpawnOnMaxReconnect = true
	// just in case it has a proper connection, still close it.
	defer client.Close()

	if err := client.Connect(); err != nil {
		return err
	}
	return execFn(client)
}

func SetDefaultPort(port string) {
	currentPort = port
}

func CurrentPort() string {
	return ":" + currentPort
}
