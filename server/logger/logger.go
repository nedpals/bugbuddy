package logger

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lucasepe/codename"

	_ "embed"

	"github.com/nedpals/bugbuddy/server/helpers"
	_ "modernc.org/sqlite"
)

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	t, err := time.Parse(time.RFC3339Nano, value.(string))
	if err != nil {
		return nil
	}

	nt.Time = t
	nt.Valid = true
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return time.Now().Format(time.RFC3339Nano), nil
	}
	return nt.Time.Format(time.RFC3339Nano), nil
}

//go:embed init.sql
var initScript string

type Logger struct {
	participantId string
	db            *sqlx.DB
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
	// get or initialize directory
	dirPath, err := helpers.GetOrInitializeDataDir()
	if err != nil {
		return nil, err
	}

	logsDbPath := filepath.Join(dirPath, "logs.db")
	return NewLoggerFromPath(logsDbPath)
}

func NewLoggerPanic() *Logger {
	logger, err := NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewLoggerFromPath(path string) (*Logger, error) {
	if !filepath.IsAbs(path) {
		rPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		path = rPath
	}

	return setupLogger(path)
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
	db, err := sqlx.Open("sqlite", logsDbPath)
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

func (log *Logger) GetSetting(key string) (string, error) {
	var val string
	err := log.db.QueryRow("SELECT value FROM settings WHERE name = ?", key).Scan(&val)
	return val, err
}

func (log *Logger) AddSetting(key, value string) error {
	_, err := log.db.Exec("INSERT OR REPLACE INTO settings (name, value) VALUES (?, ?)", key, value)
	return err
}

func (log *Logger) DeleteSetting(key string) error {
	_, err := log.db.Exec("DELETE FROM settings WHERE name = ?", key)
	return err
}

func (log *Logger) ParticipantId() string {
	// get cached value to avoid burdening the database
	if len(log.participantId) == 0 {
		val, _ := log.GetSetting("participant_id")
		log.participantId = val
	}
	return log.participantId
}

func (log *Logger) GenerateParticipantId() error {
	// retrieve seed for rng (and generate if not found)
	// seed will be used to generate participant id
	seed := int64(0)

	if rawSeed, err := log.GetSetting("_seed"); err == nil {
		generatedSeed, err := strconv.ParseInt(rawSeed, 10, 64)
		if err != nil {
			return err
		}
		seed = generatedSeed
	} else if err == sql.ErrNoRows {
		generatedSeed, err := log.GenerateSeed()
		if err != nil {
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
		if err := log.DeleteSetting("_seed"); err != nil {
			return err
		}

		return log.GenerateParticipantId()
	}

	// add participant id
	err := log.AddSetting("participant_id", participantId)
	if err != nil {
		return err
	}

	log.participantId = ""
	return nil
}

func (log *Logger) Setup() error {
	// check if participant id has been set
	pId := log.ParticipantId()
	if len(pId) == 0 {
		return log.GenerateParticipantId()
	}
	return nil
}

func (log *Logger) GenerateSeed() (int64, error) {
	seed, err := codename.NewCryptoSeed()
	if err != nil {
		return 0, err
	}

	_ = log.AddSetting("_seed", strconv.FormatInt(seed, 10))
	return seed, nil
}

type LogEntry struct {
	Id              int       `db:"id,omitempty"`
	ParticipantId   string    `db:"participant_id"`
	ExecutedCommand string    `db:"executed_command"`
	ErrorType       string    `db:"error_type"`
	ErrorCode       int       `db:"error_code"`
	ErrorMessage    string    `db:"error_message"`
	ErrorLine       int       `db:"error_line"`
	ErrorColumn     int       `db:"error_column"`
	GeneratedOutput string    `db:"generated_output"`
	FilePath        string    `db:"file_path"`
	FileVersion     int       `db:"file_version"`
	CreatedAt       *NullTime `db:"created_at,omitempty"`
}

func (log *Logger) Log(entry LogEntry) error {
	if len(entry.ParticipantId) == 0 {
		entry.ParticipantId = log.ParticipantId()
	}

	if entry.CreatedAt == nil || !entry.CreatedAt.Valid || entry.CreatedAt.Time.IsZero() {
		entry.CreatedAt = &NullTime{Time: time.Now(), Valid: true}
	}

	_, err := log.db.NamedExec(`INSERT INTO logs (
	participant_id, executed_command, 
	error_code, error_line, error_column, error_type,
	error_message, generated_output, file_path, 
	file_version, created_at
) VALUES (
	:participant_id, :executed_command, 
	:error_code, :error_line, :error_column, :error_type,
	:error_message, :generated_output, :file_path, 
	:file_version, :created_at
)`, &entry)
	return err
}

// Implement a streaming iterator for log entries
// This will allow us to iterate through the log entries without loading all of them into memory
// This is useful for large logs
type LogEntryIterator struct {
	rows *sqlx.Rows
}

func (it *LogEntryIterator) Next() bool {
	res := it.rows.Next()
	if !res {
		it.rows.Close()
	}
	return res
}

func (it *LogEntryIterator) Value() (LogEntry, error) {
	var entry LogEntry
	if err := it.rows.StructScan(&entry); err != nil {
		it.rows.Close()
		return LogEntry{}, err
	}
	return entry, nil
}

func (it *LogEntryIterator) List() ([]LogEntry, error) {
	var entries []LogEntry
	for it.Next() {
		entry, err := it.Value()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (log *Logger) Entries() (*LogEntryIterator, error) {
	rows, err := log.db.Queryx("SELECT * FROM logs")
	if err != nil {
		defer rows.Close()
		return nil, err
	}
	return &LogEntryIterator{rows: rows}, nil
}

func (log *Logger) EntriesByParticipantId(participantId string) (*LogEntryIterator, error) {
	rows, err := log.db.Queryx("SELECT * FROM logs WHERE participant_id = ?", participantId)
	if err != nil {
		defer rows.Close()
		return nil, err
	}
	return &LogEntryIterator{rows: rows}, nil
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
func (log *Logger) OpenFile(filepath string) ([]byte, error) {
	var content []byte
	err := log.db.QueryRow("SELECT content FROM files WHERE participant_id = ? AND file_path = ?", log.ParticipantId(), filepath).Scan(&content)
	return content, err
}

func (log *Logger) OpenVersionedFile(filepath string, file_version int) ([]byte, error) {
	var content []byte
	// get the recent file by file_version
	err := log.db.QueryRow(
		"SELECT content FROM files WHERE participant_id = ? AND file_path = ? AND file_version = ?",
		log.ParticipantId(),
		filepath,
		file_version).Scan(&content)

	return content, err
}

func (log *Logger) WriteFile(filepath string, content []byte) error {
	_, err := log.db.Exec(
		"INSERT OR REPLACE INTO files (participant_id, file_path, content, created_at) VALUES (?, ?, ?, ?)",
		log.ParticipantId(),
		filepath,
		content,
		time.Now(),
	)
	return err
}

func (log *Logger) LatestVersionFromFile(filepath string) (int, error) {
	// get latest file version
	var maxVersion int

	err := log.db.QueryRow(
		"SELECT MAX(file_version) FROM files WHERE participant_id = ? AND file_path = ?",
		log.ParticipantId(),
		filepath,
	).Scan(&maxVersion)
	if err != nil {
		return -1, fmt.Errorf("we cannot get the latest file version: %w", err)
	}

	return maxVersion, nil
}

func (log *Logger) WriteVersionedFile(filepath string, content []byte, file_version int) error {
	if file_version < 0 {
		maxVersion, err := log.LatestVersionFromFile(filepath)
		if err != nil {
			return err
		}

		file_version = maxVersion + 1
	}

	_, err := log.db.Exec(
		"INSERT INTO files (participant_id, file_path, file_version, content, created_at) VALUES (?, ?, ?, ?, ?)",
		log.ParticipantId(),
		filepath,
		file_version,
		content,
		time.Now(),
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
	// add a check to avoid nil pointer dereference
	if log == nil || log.db == nil {
		return nil
	}
	return log.db.Close()
}
