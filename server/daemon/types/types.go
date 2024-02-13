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

type ErrorReport struct {
	Template      string               `json:"template"`
	Language      string               `json:"language"`
	ErrorCode     int                  `json:"exit_code"`
	Received      int                  `json:"received"`
	Processed     int                  `json:"processed"`
	AnalyzerError string               `json:"analyzer_error"`
	FullMessage   string               `json:"full_message"`
	Message       string               `json:"message"`
	Location      errgoengine.Location `json:"location"`
}

type NearestNodePayload struct {
	DocumentIdentifier
	Line   int
	Column int
}

type ServerInfo struct {
	Success                 bool     `json:"success"`
	Version                 string   `json:"version"`
	ProcessID               int      `json:"process_id"`
	SupportedFileExtensions []string `json:"supported_file_extensions"`
}

type CollectResponse struct {
	Recognized int
	Processed  int
	Error      string
}

type SetDataDirRequest struct {
	NewPath string `json:"new_path"`
}
