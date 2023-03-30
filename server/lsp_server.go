package main

import (
	"context"
	"log"
	"net"

	"github.com/sourcegraph/jsonrpc2"
)

type LspServer struct{}

func (s *LspServer) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	if err := c.Reply(ctx, r.ID, "test"); err != nil {
		log.Println(err)
		return
	}
}

var server = &LspServer{}

func acceptIncomingLspRequest(conn net.Conn) {
	srv := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
		server,
	)
	defer srv.Close()
}

func startLspServer(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go acceptIncomingLspRequest(conn)
	}
}
