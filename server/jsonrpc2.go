package main

import (
	"context"
	"net"

	"github.com/sourcegraph/jsonrpc2"
)

func startRpcServer(addr string, codec jsonrpc2.ObjectCodec, h jsonrpc2.Handler) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer l.Close()

	asyncH := jsonrpc2.AsyncHandler(h)

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		jsonrpc2.NewConn(
			context.Background(),
			jsonrpc2.NewBufferedStream(conn, codec),
			asyncH,
		)
	}
}
