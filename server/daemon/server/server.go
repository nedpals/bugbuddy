package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/nedpals/bugbuddy/server/daemon/types"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/release"
	"github.com/nedpals/bugbuddy/server/rpc"
	"github.com/nedpals/errgoengine"
	"github.com/nedpals/errgoengine/error_templates"
	"github.com/nedpals/errgoengine/languages"
	"github.com/sourcegraph/jsonrpc2"
)

type resultError struct {
	report  *types.ErrorReport
	version int // TODO: make it working
}

type Server struct {
	ServerLog *log.Logger
	engine    *errgoengine.ErrgoEngine
	// TODO: add storage for context data
	connectedClients connectedClients
	logger           *logger.Logger
	errors           []resultError
}

func (d *Server) SetLogger(l *logger.Logger) error {
	// Close the old logger before setting a new one
	if err := d.logger.Close(); err != nil {
		return err
	}

	d.logger = l
	return nil
}

func (d *Server) Clients() connectedClients {
	return d.connectedClients
}

func (d *Server) Engine() *errgoengine.ErrgoEngine {
	return d.engine
}

func (d *Server) FS() *helpers.SharedFS {
	return d.engine.FS.FSs[0].(*helpers.SharedFS)
}

func (d *Server) getProcessId(r *jsonrpc2.Request) (int, error) {
	for _, req := range r.ExtraFields {
		if req.Name != "processId" {
			continue
		}
		procId := req.Value.(json.Number)
		num, err := procId.Int64()
		if err != nil {
			break
		}
		return int(num), nil
	}
	return -1, errors.New("processId not found")
}

func (d *Server) checkProcessConnection(r *jsonrpc2.Request) *jsonrpc2.Error {
	procId, err := d.getProcessId(r)
	if err != nil {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process ID not found",
		}
	}

	if _, found := d.connectedClients[procId]; !found {
		return &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidRequest,
			Message: "Process not connected yet.",
		}
	}

	return nil
}

func (d *Server) Handle(ctx context.Context, c *jsonrpc2.Conn, r *jsonrpc2.Request) {
	if !types.MethodIsEither(r.Method, types.HandshakeMethod, types.ShutdownMethod) {
		if err := d.checkProcessConnection(r); err != nil {
			c.ReplyWithError(ctx, r.ID, err)
			return
		}
	}

	switch types.Method(r.Method) {
	case types.HandshakeMethod:
		// TODO: add checks and result
		var info types.ClientInfo
		if err := json.Unmarshal(*r.Params, &info); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		} else if info.ClientType < types.MonitorClientType || info.ClientType >= types.UnknownClientType {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unknown client type.",
			})
			return
		}

		d.ServerLog.Printf("connected: {process_id: %d, type: %d}\n", info.ProcessId, info.ClientType)
		d.connectedClients[info.ProcessId] = connectedClient{
			id:         info.ProcessId,
			clientType: info.ClientType,
			conn:       c,
		}

		engineSupportedExtensions := []string{}
		for _, lang := range languages.SupportedLanguages {
			engineSupportedExtensions = append(engineSupportedExtensions, lang.FilePatterns...)
		}

		// introduce the server to the client
		c.Reply(ctx, r.ID, &types.ServerInfo{
			Success:                 true,
			Version:                 release.Version(),
			ProcessID:               info.ProcessId,
			SupportedFileExtensions: engineSupportedExtensions,
		})

		// Send the existing errors to a newly connected client
		if info.ClientType == types.LspClientType {
			d.notifyErrors(ctx, d.errors, info.ProcessId)
		}
	case types.ShutdownMethod:
		procId, err := d.getProcessId(r)
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}

		delete(d.connectedClients, procId)
		d.ServerLog.Printf("disconnected: {process_id: %d}\n", procId)
	case types.CollectMethod:
		var payload types.CollectPayload
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
		}

		rec, p, err := d.collect(ctx, payload, c)
		if err != nil {
			d.ServerLog.Printf("collect error: %s\n", err.Error())
			c.Reply(ctx, r.ID, types.CollectResponse{
				Recognized: rec,
				Processed:  p,
				Error:      err.Error(),
			})
		} else {
			c.Reply(ctx, r.ID, types.CollectResponse{
				Recognized: rec,
				Processed:  p,
				Error:      "",
			})
		}
	case types.PingMethod:
		procId, err := d.getProcessId(r)
		if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}
		d.ServerLog.Printf("ping from %d\n", procId)
		c.Reply(ctx, r.ID, "pong!")
	case types.ResolveDocumentMethod:
		var payloadStr types.DocumentPayload
		if err := json.Unmarshal(*r.Params, &payloadStr); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		if len(payloadStr.Filepath) == 0 {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Filepath is empty",
			})
			return
		}

		if err := d.FS().WriteFile(payloadStr.Filepath, []byte(payloadStr.Content)); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}

		d.ServerLog.Printf("resolved document: %s (len: %d)\n", payloadStr.Filepath, len(payloadStr.Content))
		c.Reply(ctx, r.ID, "ok")
	case types.UpdateDocumentMethod:
		var payloadStr types.DocumentPayload
		if err := json.Unmarshal(*r.Params, &payloadStr); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		if len(payloadStr.Filepath) == 0 {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Filepath is empty",
			})
			return
		}

		// check if the file exists
		if file, err := d.FS().Open(payloadStr.Filepath); errors.Is(err, fs.ErrNotExist) {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "File does not exist",
			})
			return
		} else if err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		} else {
			file.Close()
		}

		// IDEA: create a dependency tree wherein errors will be removed
		// once the file is updated
		if err := d.FS().WriteFile(payloadStr.Filepath, []byte(payloadStr.Content)); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}

		d.ServerLog.Printf("updated document: %s (len: %d)\n", payloadStr.Filepath, len(payloadStr.Content))
		c.Reply(ctx, r.ID, "ok")
	case types.DeleteDocumentMethod:
		var payload types.DocumentIdentifier
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		if len(payload.Filepath) == 0 {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Filepath is empty",
			})
			return
		} else if file, err := d.FS().Open(payload.Filepath); errors.Is(err, fs.ErrNotExist) {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "File does not exist",
			})
			return
		} else {
			file.Close()
		}

		// TODO: use dependency tree
		if err := d.FS().Remove(payload.Filepath); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}

		d.ServerLog.Printf("removed document: %s\n", payload.Filepath)
		c.Reply(ctx, r.ID, "ok")

		// doc := d.engine
	case types.RetrieveParticipantIdMethod:
		c.Reply(ctx, r.ID, d.logger.ParticipantId())
	case types.GenerateParticipantIdMethod:
		if err := d.logger.GenerateParticipantId(); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}
		c.Reply(ctx, r.ID, d.logger.ParticipantId())
	case types.ResetLoggerMethod:
		if err := d.logger.Reset(); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: err.Error(),
			})
			return
		}
		c.Reply(ctx, r.ID, "ok")
	case types.GetDataDirMethod:
		dataDir := helpers.GetDataDirPath()
		c.Reply(ctx, r.ID, dataDir)
	case types.SetDataDirMethod:
		var payload types.SetDataDirRequest
		if err := json.Unmarshal(*r.Params, &payload); err != nil {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "Unable to decode params of method " + r.Method,
			})
			return
		}

		if len(payload.NewPath) == 0 {
			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: "New path must not be empty",
			})
			return
		}

		// get the old data dir
		oldDataDir := helpers.GetDataDirPath()

		// set the new data dir
		helpers.SetDataDirPath(payload.NewPath)

		// reload the logger. we wont be using NewLoggerPanic because
		// we dont want to crash the daemon if the logger fails to load
		logger, err := logger.NewLogger()
		if err != nil {
			if logger != nil {
				// close the logger just to be sure
				logger.Close()
			}

			// revert the data dir
			helpers.SetDataDirPath(oldDataDir)

			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: fmt.Sprintf("Something went wrong while setting the new data dir: %s", err.Error()),
			})
			return
		}

		if err := d.SetLogger(logger); err != nil {
			if logger != nil {
				// close the logger if it's not nil
				logger.Close()
			}

			// revert the data dir
			helpers.SetDataDirPath(oldDataDir)

			c.ReplyWithError(ctx, r.ID, &jsonrpc2.Error{
				Message: fmt.Sprintf("Something went wrong while setting the new data dir: %s", err.Error()),
			})
			return
		}

		d.ServerLog.Printf("data dir set to %s\n", payload.NewPath)
		c.Reply(ctx, r.ID, "ok")
	}
}

func (s *Server) collect(ctx context.Context, payload types.CollectPayload, c *jsonrpc2.Conn) (recognized int, processed int, err error) {
	result := helpers.AnalyzeError(s.engine, payload.WorkingDir, payload.Error)
	r, p, err := result.Stats()
	s.ServerLog.Printf("collect: %d recognized, %d processed\n", r, p)
	if len(payload.Error) == 0 && payload.ErrorCode < 1 {
		s.notifyErrors(ctx, []resultError{
			{
				report: &types.ErrorReport{
					Message:       result.Exp,
					ErrorCode:     payload.ErrorCode,
					Received:      r,
					Processed:     p,
					AnalyzerError: "",
				},
			},
		})
	}

	if err != nil {

		return r, p, err
	}

	logPayload := logger.LogEntry{
		ExecutedCommand: payload.Command,
		ErrorCode:       payload.ErrorCode,
		ErrorMessage:    payload.Error,
		FilePath:        result.Data.MainError.Document.Path,
		GeneratedOutput: result.Output,
	}

	analyzerError := ""
	if err != nil {
		analyzerError = err.Error()
	}

	report := resultError{
		report: &types.ErrorReport{
			FullMessage:   result.Output,
			Message:       result.Exp,
			Template:      result.Template.Name,
			Language:      result.Template.Language.Name,
			Location:      result.Data.MainError.Nearest.Location(),
			ErrorCode:     payload.ErrorCode,
			Received:      r,
			Processed:     p,
			AnalyzerError: analyzerError,
		},
	}

	s.logger.Log(logPayload)
	s.errors = append(s.errors, report)
	s.notifyErrors(ctx, []resultError{report})

	// write files to the logger
	for _, file := range result.Data.Documents {
		s.logger.WriteFile(file.Path, []byte(file.Contents))
	}

	return r, p, nil
}

func (s *Server) notifyErrors(ctx context.Context, errors []resultError, procIds_ ...int) {
	s.ServerLog.Printf("report %d error/s to %d clients\n", len(errors), len(s.connectedClients.ProcessIds(types.LspClientType)))

	lspClients := procIds_
	if len(lspClients) == 0 {
		lspClients = s.connectedClients.ProcessIds(types.LspClientType)
	}

	// TODO: cleanup old errors
	for _, r := range errors {
		// TODO: make sure that the errors sent are within their working dir
		s.connectedClients.Notify(ctx, types.ReportMethod, r.report, lspClients...)
	}
}

func (s *Server) Start(addr string) error {
	return rpc.StartServer(
		addr,
		jsonrpc2.VarintObjectCodec{},
		s,
	)
}

func NewServer() *Server {
	server := &Server{
		ServerLog: log.New(os.Stdout, "server> ", 0),
		engine: &errgoengine.ErrgoEngine{
			ErrorTemplates: errgoengine.ErrorTemplates{},
			FS: &errgoengine.MultiReadFileFS{
				FSs: []fs.ReadFileFS{
					helpers.NewSharedFS(),
				},
			},
			SharedStore: errgoengine.NewEmptyStore(),
			OutputGen:   &errgoengine.OutputGenerator{},
		},
		connectedClients: connectedClients{},
		errors:           []resultError{},
		logger:           logger.NewMemoryLoggerPanic(),
	}

	error_templates.LoadErrorTemplates(&server.engine.ErrorTemplates)
	return server
}

func Start(server *Server, addr string) error {
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	errChan := make(chan error, 1)
	disconnChan := make(chan int, 1)
	exitSignal := make(chan os.Signal, 1)

	if err := server.SetLogger(logger.NewLoggerPanic()); err != nil {
		fmt.Printf("logger error: %s\n", err.Error())
	}

	go func() {
		fmt.Println("daemon started on " + addr)
		errChan <- server.Start(addr)
	}()

	signal.Notify(exitSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSignal
		disconnChan <- 1
	}()

	for {
		select {
		case err := <-errChan:
			return err
		case <-time.After(15 * time.Second):
			// Disconnect only if CTRL+C is pressed or is launched
			// as a background terminal
			if !isTerminal && len(server.connectedClients) == 0 {
				disconnChan <- 1
			}
		case <-disconnChan:
			server.connectedClients.Disconnect()
			return nil
		}
	}
}
