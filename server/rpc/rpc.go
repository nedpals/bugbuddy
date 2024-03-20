package rpc

import (
	"context"
	"io"
	"net"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

type HandlerFunc func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request)

func (h HandlerFunc) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	h(ctx, c, r)
}

type CustomStream struct {
	io.ReadCloser
	io.WriteCloser
}

func (conn *CustomStream) Read(p []byte) (n int, err error) {
	return conn.ReadCloser.Read(p)
}

func (conn *CustomStream) Write(p []byte) (n int, err error) {
	return conn.WriteCloser.Write(p)
}

func (conn *CustomStream) Close() error {
	if err := conn.ReadCloser.Close(); err != nil {
		return err
	} else if err := conn.WriteCloser.Close(); err != nil {
		return err
	}
	return nil
}

func StartServer(addr string, codec jsonrpc2.ObjectCodec, h jsonrpc2.Handler) error {
	if strings.HasPrefix(addr, ":") {
		// Prepend 127.0.0.1 to the address. This is for us to be able to
		// run the server without firewall prompts. (for Windows)
		// https://stackoverflow.com/a/66486551
		addr = "127.0.0.1" + addr
	}

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

		go func() {
			cn := jsonrpc2.NewConn(
				context.Background(),
				jsonrpc2.NewBufferedStream(conn, codec),
				asyncH,
			)
			defer cn.Close()
			<-cn.DisconnectNotify()
		}()
	}
}
