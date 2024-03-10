package logger_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/nedpals/bugbuddy/server/logger"
)

func TestLogger_Log(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Test Log
	params := logger.LogEntry{
		ExecutedCommand: "go test",
		ErrorCode:       1,
		ErrorMessage:    "Test failed",
		GeneratedOutput: "Some output",
		FilePath:        "/path/to/file.go",
	}
	err = log.Log(params)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogger_Entries(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Add multiple logs
	for i := 0; i < 5; i++ {
		params := logger.LogEntry{
			ExecutedCommand: "go test",
			ErrorCode:       1,
			ErrorMessage:    "Test failed",
			GeneratedOutput: "Some output",
			FilePath:        fmt.Sprintf("/path/to/file%d.go", i),
		}

		err = log.Log(params)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test Entries
	entriesIter, err := log.Entries()
	if err != nil {
		t.Fatal(err)
	}

	entries, err := entriesIter.List()
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved entries with the original entries
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestLogger_EntriesByParticipantId(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Add multiple logs (first loop is the participant id and the second loop is the log entry)
	participantIds := []string{
		log.ParticipantId(),
		"participant2",
	}

	for _, participantId := range participantIds {
		for i := 0; i < 5; i++ {
			params := logger.LogEntry{
				ParticipantId:   participantId,
				ExecutedCommand: "go test",
				ErrorCode:       1,
				ErrorMessage:    "Test failed",
				GeneratedOutput: "Some output",
				FilePath:        fmt.Sprintf("/path/to/file%d.go", i),
			}

			err = log.Log(params)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test EntriesByParticipantId
	entriesIter, err := log.EntriesByParticipantId(log.ParticipantId())
	if err != nil {
		t.Fatal(err)
	}

	entries, err := entriesIter.List()
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved entries with the original entries
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestLogger_AddSetting(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Test AddSetting
	err = log.AddSetting("key", "value")
	if err != nil {
		t.Fatal(err)
	}

	// Test GetSetting
	value, err := log.GetSetting("key")
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved value with the original value
	if value != "value" {
		t.Errorf("expected value to be value, got %s", value)
	}
}

func TestLogger_GetSetting(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Test AddSetting
	err = log.AddSetting("key", "value")
	if err != nil {
		t.Fatal(err)
	}

	// Test GetSetting
	value, err := log.GetSetting("key")
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved value with the original value
	if value != "value" {
		t.Errorf("expected value to be value, got %s", value)
	}
}

func TestLogger_DeleteSetting(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Test AddSetting
	err = log.AddSetting("key", "value")
	if err != nil {
		t.Fatal(err)
	}

	// Test DeleteSetting
	err = log.DeleteSetting("key")
	if err != nil {
		t.Fatal(err)
	}

	// Test GetSetting
	_, err = log.GetSetting("key")
	if err == nil {
		t.Errorf("expected setting to be deleted")
	}
}

func TestLogger_OpenFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteFile(filePath, content)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	retrievedContent, err := log.OpenFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved content with the original content
	if !bytes.Equal(content, retrievedContent) {
		t.Errorf("expected file content to be %s, got %s", string(content), string(retrievedContent))
	}
}

func TestLogger_OpenVersionedFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteVersionedFile(filePath, content, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	retrievedContent, err := log.OpenVersionedFile(filePath, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved content with the original content
	if !bytes.Equal(content, retrievedContent) {
		t.Errorf("expected file content to be %s, got %s", string(content), string(retrievedContent))
	}
}

func TestLogger_WriteFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteFile(filePath, content)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	retrievedContent, err := log.OpenFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved content with the original content
	if !bytes.Equal(content, retrievedContent) {
		t.Errorf("expected file content to be %s, got %s", string(content), string(retrievedContent))
	}
}

func TestLogger_WriteVersionedFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteVersionedFile(filePath, content, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	retrievedContent, err := log.OpenVersionedFile(filePath, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved content with the original content
	if !bytes.Equal(content, retrievedContent) {
		t.Errorf("expected file content to be %s, got %s", string(content), string(retrievedContent))
	}
}

func TestLogger_RenameFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteFile(filePath, content)
	if err != nil {
		t.Fatal(err)
	}

	// Generate a new file path
	newFilePath := "/path/to/new_file.go"

	// Rename the file
	err = log.RenameFile(filePath, newFilePath)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	retrievedContent, err := log.OpenFile(newFilePath)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the retrieved content with the original content
	if !bytes.Equal(content, retrievedContent) {
		t.Errorf("expected file content to be %s, got %s", string(content), string(retrievedContent))
	}
}

func TestLogger_DeleteFile(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteFile(filePath, content)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the file
	err = log.DeleteFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	_, err = log.OpenFile(filePath)
	if err == nil {
		t.Errorf("expected file to be deleted")
	}
}

func TestLogger_Reset(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a random file path
	filePath := "/path/to/file.go"

	// Generate random file content
	content := []byte("package main\n\nfunc main() {\n\t// Code here\n}")

	// Write the file
	err = log.WriteFile(filePath, content)
	if err != nil {
		t.Fatal(err)
	}

	// Reset the logger
	err = log.Reset()
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the file content
	_, err = log.OpenFile(filePath)
	if err == nil {
		t.Errorf("expected file to be deleted")
	}
}

func TestLogger_Close(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}

	// Close the logger
	err = log.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Test if the logger is closed
	_, err = log.OpenFile("/path/to/file.go")
	if err == nil {
		t.Errorf("expected logger to be closed")
	}
}

func TestLogger_CloseNullLogger(t *testing.T) {
	// Create a new logger
	var log *logger.Logger

	// Close the logger
	err := log.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogger_GenerateParticipantIdExistingSeed(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Add a seed
	err = log.AddSetting("_seed", "12121111")
	if err != nil {
		t.Fatal(err)
	}

	// check if seed exists
	seed, err := log.GetSetting("_seed")
	if err != nil {
		t.Fatal(err)
	}

	// Generate a participant ID
	if err := log.GenerateParticipantId(); err != nil {
		t.Fatal(err)
	}

	participantId := log.ParticipantId()

	fmt.Println(seed)
	if len(seed) == 0 {
		t.Fatalf("expected seed to be generated")
	}

	// Generate a new participant ID
	if err := log.GenerateParticipantId(); err != nil {
		t.Fatal(err)
	}

	newParticipantId := log.ParticipantId()

	// Compare the participant IDs
	if participantId == newParticipantId {
		t.Errorf("expected participant IDs to be different")
	}
}

func TestLogger_GenerateParticipantIdInvalidSeed(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Add a seed
	err = log.AddSetting("_seed", "seed")
	if err != nil {
		t.Fatal(err)
	}

	// check if seed exists
	seed, err := log.GetSetting("_seed")
	if err != nil {
		t.Fatal(err)
	} else if seed != "seed" {
		t.Fatalf("expected seed to be seed")
	}

	// Generate a participant ID
	if err := log.GenerateParticipantId(); err == nil {
		t.Fatalf("expected error to be thrown")
	} else if err.Error() != "strconv.ParseInt: parsing \"seed\": invalid syntax" {
		t.Fatalf("expected error to be strconv.ParseInt: parsing \"seed\": invalid syntax")
	}
}

func TestLogger_GenerateParticipantIdReset(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	// Generate a participant ID
	if err := log.GenerateParticipantId(); err != nil {
		t.Fatal(err)
	}

	participantId := log.ParticipantId()

	// Reset the logger
	err = log.Reset()
	if err != nil {
		t.Fatal(err)
	}

	// Generate a new participant ID
	if err := log.GenerateParticipantId(); err != nil {
		t.Fatal(err)
	}

	newParticipantId := log.ParticipantId()

	// Compare the participant IDs
	if participantId == newParticipantId {
		t.Errorf("expected participant IDs to be different")
	}
}

func TestNewMemoryLogger(t *testing.T) {
	// Create a new memory logger
	_, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewLoggerFromPath(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	_, err := logger.NewLoggerFromPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewMemoryLoggerPanic(t *testing.T) {
	// Create a new memory logger
	_ = logger.NewMemoryLoggerPanic()
}

func TestNewLoggerFromPathPanic(t *testing.T) {
	// Create a temporary database file for testing
	dbPath := "test.db"
	defer os.Remove(dbPath)

	// Create a new logger
	_ = logger.NewLoggerFromPathPanic(dbPath)
}

func TestNewLoggerPanic(t *testing.T) {
	// Create a new logger
	_ = logger.NewLoggerPanic()
}
