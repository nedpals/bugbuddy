package rpc

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	*jsonrpc2.Conn
}

func (c *Client) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {

}

func (c *Client) Call(method string, payload any, result any) error {
	return c.Conn.Call(context.Background(), method, payload, result)
}

func (c *Client) Notify(method string, payload any) error {
	return c.Conn.Notify(context.Background(), method, payload)
}
