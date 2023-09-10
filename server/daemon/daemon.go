package daemon

import (
	"context"

	"github.com/nedpals/bugbuddy/server/daemon/client"
	"github.com/nedpals/bugbuddy/server/daemon/server"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/rpc"
)

const DEFAULT_PORT = ":3434"

func NewClient(ctx context.Context, addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *client.Client {
	return client.NewClient(ctx, addr, clientType, handlerFunc...)
}

func Connect(addr string, clientType types.ClientType, handlerFunc ...rpc.HandlerFunc) *client.Client {
	return client.Connect(addr, clientType, handlerFunc...)
}

func Serve(addr string) error {
	return server.Start(addr)
}
