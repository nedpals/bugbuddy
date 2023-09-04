package server

import (
	"context"
	"errors"

	"github.com/nedpals/bugbuddy-proto/server/daemon/types"
	"github.com/sourcegraph/jsonrpc2"
)

type connectedClient struct {
	id         int
	clientType types.ClientType
	conn       *jsonrpc2.Conn
}

type connectedClients map[int]connectedClient

func (clients connectedClients) Notify(ctx context.Context, method types.Method, params any, clientTypes ...types.ClientType) error {
	var errs []error

	for _, c := range clients {
		if len(clientTypes) != 0 {
			shouldBeNotified := false

			for _, ct := range clientTypes {
				if c.clientType == ct {
					shouldBeNotified = true
				}
			}

			if !shouldBeNotified {
				continue
			}
		}

		if err := c.conn.Notify(ctx, string(method), params); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (clients connectedClients) Disconnect() {
	for _, cl := range clients {
		cl.conn.Close()
		delete(clients, cl.id)
	}
}
