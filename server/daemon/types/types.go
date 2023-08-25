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

type CollectPayload struct {
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
	Message string
}
