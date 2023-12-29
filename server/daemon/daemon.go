package daemon

import (
	"context"
	"os"

	"github.com/nedpals/bugbuddy/server/daemon/client"
	"github.com/nedpals/bugbuddy/server/daemon/server"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
)

type Client = client.Client

const DEFAULT_PORT = ":3434"

var CURRENT_DAEMON_PORT = DEFAULT_PORT

func init() {
	if port := os.Getenv("BUGBUDDY_DAEMON_PORT"); len(port) > 0 {
		CURRENT_DAEMON_PORT = ":" + port
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
	client := NewClient(context.Background(), CURRENT_DAEMON_PORT, clientType)
	if err := client.Connect(); err != nil {
		if err := client.EnsureConnection(); err != nil {
			return err
		}
	}
	defer client.Close()
	return execFn(client)
}
