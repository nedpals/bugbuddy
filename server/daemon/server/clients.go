package server

import (
	"context"
	"errors"

	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/sourcegraph/jsonrpc2"
)

type connectedClient struct {
	id         int
	clientType types.ClientType
	conn       *jsonrpc2.Conn
}

type connectedClients map[int]connectedClient

func (clients connectedClients) ProcessIds(clientTypes ...types.ClientType) []int {
	procIds := []int{}

	for _, cl := range clients {
		for _, ct := range clientTypes {
			if cl.clientType == ct {
				procIds = append(procIds, cl.id)
				break
			}
		}
	}

	return procIds
}

func (clients connectedClients) Notify(ctx context.Context, method types.Method, params any, procIds ...int) error {
	var errs []error

	if len(procIds) != 0 {
		for _, procId := range procIds {
			c, ok := clients[procId]
			if !ok {
				continue
			} else if err := c.conn.Notify(ctx, string(method), params); err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		for _, c := range clients {
			if err := c.conn.Notify(ctx, string(method), params); err != nil {
				errs = append(errs, err)
			}
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

func (clients connectedClients) Get(procId int) (int, types.ClientType) {
	c, ok := clients[procId]
	if ok {
		return c.id, c.clientType
	}
	return -1, types.UnknownClientType
}
