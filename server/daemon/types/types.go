package types

import "github.com/nedpals/errgoengine"

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

type CollectPayload struct {
	ErrorCode  int
	Command    string
	Error      string
	WorkingDir string
}

type DocumentIdentifier struct {
	Filepath string `json:"filepath"`
}

type DocumentPayload struct {
	DocumentIdentifier
	Content string `json:"content"`
}

// TODO: dummy payload for now. should give back instructions instead of the error message
type ErrorReport struct {
	Template string               `json:"template"`
	Language string               `json:"language"`
	Message  string               `json:"message"`
	Location errgoengine.Location `json:"location"`
}

type NearestNodePayload struct {
	DocumentIdentifier
	Line   int
	Column int
}
