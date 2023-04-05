package types

type ClientType int

const (
	MonitorClientType ClientType = 0
	LspClientType     ClientType = iota
	UnknownClientType ClientType = iota
)

type ClientInfo struct {
	ProcessId  int        `json:"processId"`
	ClientType ClientType `json:"clientType"`
}

// TODO: dummy payload for now. should give back instructions instead of the error message
type ErrorReport struct {
	Message string
}
