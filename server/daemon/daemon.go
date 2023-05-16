package daemon

import (
	"context"

	"github.com/nedpals/bugbuddy-proto/server/daemon/client"
	"github.com/nedpals/bugbuddy-proto/server/daemon/server"
	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/sourcegraph/jsonrpc2"
)

const DEFAULT_PORT = ":3434"

func NewClient(addr string, clientType types.ClientType, handlerFunc ...func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request)) *client.Client {
	return client.NewClient(addr, clientType, handlerFunc...)
}

func Connect(addr string, clientType types.ClientType, handlerFunc ...func(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request)) *client.Client {
	return client.Connect(addr, clientType, handlerFunc...)
}

func Serve(addr string) error {
	return server.Start(addr)
}
