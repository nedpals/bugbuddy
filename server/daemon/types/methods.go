package types

import "fmt"

// for building consistent jsonrpc method proc names
type namespace string

type Method string

func (n namespace) methodName(method string) Method {
	return Method(fmt.Sprintf("%s/%s", n, method))
}

const (
	serverNamespace  = namespace("$")
	clientsNamespace = namespace("clients")
)

// server methods
var (
	HandshakeMethod = serverNamespace.methodName("handshake")
	ShutdownMethod  = serverNamespace.methodName("shutdown")
	CollectMethod   = serverNamespace.methodName("collect")
	PingMethod      = serverNamespace.methodName("ping")
)

var (
	ReportMethod = clientsNamespace.methodName("collect")
)

func MethodIs(s string, m Method) bool {
	return s == string(m)
}

func MethodIsEither(s string, ms ...Method) bool {
	for _, m := range ms {
		if MethodIs(s, m) {
			return true
		}
	}
	return false
}
