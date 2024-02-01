package logger

import (
	"database/sql"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lucasepe/codename"

	_ "embed"

	"github.com/nedpals/bugbuddy/server/helpers"
	_ "modernc.org/sqlite"
)

//go:embed init.sql
var initScript string

type Logger struct {
	participantId string
	db            *sql.DB
}

func NewMemoryLogger() (*Logger, error) {
	return setupLogger(":memory:")
}

func NewMemoryLoggerPanic() *Logger {
	logger, err := NewMemoryLogger()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewLogger() (*Logger, error) {
	return NewLoggerFromPath("logs.db")
}

func NewLoggerPanic() *Logger {
	logger, err := NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewLoggerFromPath(path string) (*Logger, error) {
	// get or initialize directory
	dirPath, err := helpers.GetOrInitializeDir()
	if err != nil {
		return nil, err
	}

	logsDbPath := filepath.Join(dirPath, path)
	return setupLogger(logsDbPath)
}

func NewLoggerFromPathPanic(path string) *Logger {
	logger, err := NewLoggerFromPath(path)
	if err != nil {
		panic(err)
	}
	return logger
}

func setupLogger(logsDbPath string) (*Logger, error) {
	// open database
	db, err := sql.Open("sqlite", logsDbPath)
	if err != nil {
		return nil, err
	}

	// initialize database
	db.Exec(initScript)
	logger := &Logger{db: db}

	if err := logger.Setup(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (log *Logger) getSetting(key string) (string, error) {
	var val string
	err := log.db.QueryRow("SELECT value FROM settings WHERE name = ?", key).Scan(&val)
	return val, err
}

func (log *Logger) addSetting(key, value string) error {
	_, err := log.db.Exec("INSERT OR REPLACE INTO settings (name, value) VALUES (?, ?)", key, value)
	return err
}

func (log *Logger) deleteSetting(key string) error {
	_, err := log.db.Exec("DELETE FROM settings WHERE name = ?", key)
	return err
}

func (log *Logger) ParticipantId() string {
	// get cached value to avoid burdening the database
	if len(log.participantId) != 0 {
		return log.participantId
	}

	val, _ := log.getSetting("participant_id")
	return val
}

func (log *Logger) GenerateParticipantId() error {
	// retrieve seed for rng (and generate if not found)
	// seed will be used to generate participant id
	seed := int64(0)

	if rawSeed, err := log.getSetting("_seed"); err == nil {
		generatedSeed, err := strconv.ParseInt(rawSeed, 10, 64)
		if err != nil {
			return err
		}
		seed = generatedSeed
	} else if err == sql.ErrNoRows {
		generatedSeed, err := codename.NewCryptoSeed()
		if err != nil {
			return err
		}

		// we store the seed to avoid collisions when generating a new participant id
		if err := log.addSetting("_seed", strconv.FormatInt(generatedSeed, 10)); err != nil {
			return err
		}

		seed = generatedSeed
	} else {
		return err
	}

	rng := rand.New(rand.NewSource(seed))
	participantId := codename.Generate(rng, 4)
	if participantId == log.participantId {
		// reset seed and try again
		if err := log.deleteSetting("_seed"); err != nil {
			return err
		}

		return log.GenerateParticipantId()
	}

	// add participant id
	err := log.addSetting("participant_id", participantId)
	if err != nil {
		return err
	}

	log.participantId = participantId
	return nil
}

func (log *Logger) Setup() error {
	// check if participant id has been set
	if _, err := log.getSetting("participant_id"); err != nil {
		if err == sql.ErrNoRows {
			return log.GenerateParticipantId()
		}
		return err
	}

	return nil
}

type LogParams struct {
	ExecutedCommand string
	ErrorCode       int
	ErrorMessage    string
	GeneratedOutput string
	FilePath        string
}

func (log *Logger) Log(params LogParams) error {
	_, err := log.db.Exec(
		"INSERT INTO logs (participant_id, executed_command, error_code, error_message, generated_output, file_path, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		log.ParticipantId(),
		params.ExecutedCommand,
		params.ErrorCode,
		params.ErrorMessage,
		params.GeneratedOutput,
		params.FilePath,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func (log *Logger) Reset() error {
	// delete logs
	if _, err := log.db.Exec("DELETE FROM logs WHERE participant_id = ?", log.ParticipantId()); err != nil {
		return err
	}

	// delete files
	if _, err := log.db.Exec("DELETE FROM files WHERE participant_id = ?", log.ParticipantId()); err != nil {
		return err
	}

	return nil
}

// logger as FS
func (log *Logger) WriteFile(filepath string, content []byte) error {
	_, err := log.db.Exec(
		"INSERT OR REPLACE INTO files (participant_id, file_path, content) VALUES (?, ?, ?)",
		log.ParticipantId(),
		filepath,
		content,
	)
	return err
}

func (log *Logger) RenameFile(oldFilepath, newFilepath string) error {
	_, err := log.db.Exec(
		"UPDATE files SET file_path = ? WHERE participant_id = ? AND file_path = ?",
		newFilepath,
		log.ParticipantId(),
		oldFilepath,
	)
	return err
}

func (log *Logger) DeleteFile(filepath string) error {
	_, err := log.db.Exec(
		"DELETE FROM files WHERE participant_id = ? AND file_path = ?",
		log.ParticipantId(),
		filepath,
	)
	return err
}

func (log *Logger) Close() error {
	return log.db.Close()
}
