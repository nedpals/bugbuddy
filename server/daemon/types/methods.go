package types

import (
	"fmt"
	"strings"
)

// for building consistent jsonrpc method proc names
type namespace string

type Method string

func (n namespace) methodName(method string) Method {
	if strings.HasSuffix(string(n), "/") {
		return Method(string(n) + method)
	}
	return Method(fmt.Sprintf("%s/%s", n, method))
}

const (
	serverNamespace    = namespace("$")
	documentsNamespace = namespace("$/documents")
	lspNamespace       = namespace("$/lsp")
	clientsNamespace   = namespace("clients")
)

// server methods
var (
	HandshakeMethod = serverNamespace.methodName("handshake")
	ShutdownMethod  = serverNamespace.methodName("shutdown")
	CollectMethod   = serverNamespace.methodName("collect")
	PingMethod      = serverNamespace.methodName("ping")
)

// document methods
var (
	ResolveDocumentMethod = documentsNamespace.methodName("resolve")
	UpdateDocumentMethod  = documentsNamespace.methodName("update")
	DeleteDocumentMethod  = documentsNamespace.methodName("delete")
)

// client methods
var (
	ReportMethod = clientsNamespace.methodName("report")
)

// lsp-specific methods
var (
	NearestNodeMethod = lspNamespace.methodName("nearestNode")
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
