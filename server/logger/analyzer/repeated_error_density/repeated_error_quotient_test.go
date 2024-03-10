package repeatederrordensity_test

import (
	"testing"

	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/logger/analyzer"
	red "github.com/nedpals/bugbuddy/server/logger/analyzer/repeated_error_density"
)

// TestREDAnalyzer tests the RED Analyzer with a set of mock error events.
func TestREDAnalyzer(t *testing.T) {
	// Setup logger
	log := logger.NewMemoryLoggerPanic()

	// Assume some participants and their compilation attempts
	mockData := []struct {
		filePath string
		events   []struct {
			errorType   string
			errorCode   int
			fileVersion int
		}
		expectedRED float64
	}{
		{
			filePath: "/test/file1.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"error1", 1, 1},
			},
			expectedRED: 0,
		},
		{
			filePath: "/test/file2.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error2", 1, 2},
				{"", 0, 3},
				{"error2", 1, 4},
				{"", 0, 5},
			},
			expectedRED: 0,
		},
		{
			filePath: "/test/file3.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error3", 1, 2},
				{"error3", 1, 3},
				{"", 0, 4},
			},
			expectedRED: 0.5,
		},
		{
			filePath: "/test/file4.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error4", 1, 2},
				{"error4", 1, 3},
				{"", 0, 4},
				{"error4", 1, 5},
				{"error4", 1, 6},
				{"", 0, 7},
			},
			expectedRED: 1,
		},
		{
			filePath: "/test/file5.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error5", 1, 2},
				{"error5", 1, 3},
				{"error5", 1, 4},
				{"", 0, 5},
			},
			expectedRED: 1.33333333,
		},
		{
			filePath: "/test/file6.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error6", 1, 2},
				{"error6", 1, 3},
				{"", 0, 4},
				{"error6", 1, 5},
				{"error6", 1, 6},
				{"", 0, 7},
				{"error6", 1, 8},
				{"error6", 1, 9},
				{"", 0, 10},
			},
			expectedRED: 1.5,
		},
		{
			filePath: "/test/file7.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error7", 1, 2},
				{"error7", 1, 3},
				{"error7", 1, 4},
				{"", 0, 5},
				{"error7", 1, 6},
				{"error7", 1, 7},
				{"", 0, 11},
			},
			expectedRED: 1.83333,
		},
		{
			filePath: "/test/file8.go",
			events: []struct {
				errorType   string
				errorCode   int
				fileVersion int
			}{
				{"", 0, 1},
				{"error8", 1, 2},
				{"error8", 1, 3},
				{"error8", 1, 4},
				{"error8", 1, 5},
				{"", 0, 6},
			},
			expectedRED: 2.25,
		},
	}

	// Log the events for each participant
	for _, p := range mockData {
		for _, e := range p.events {
			logEntry := logger.LogEntry{
				ErrorType:   e.errorType,
				ErrorCode:   e.errorCode,
				FilePath:    p.filePath,
				FileVersion: e.fileVersion,
			}
			if err := log.Log(logEntry); err != nil {
				t.Fatalf("Failed to log entry for participant: %v", err)
			}
		}
	}

	// Initialize the RED Analyzer
	redAnalyzer := analyzer.New[red.Analyzer](analyzer.LoadFromExistingLogger(log))

	// Analyze to calculate RED
	if err := redAnalyzer.Analyze(); err != nil {
		t.Fatalf("RED Analyzer failed: %v", err)
	}

	// Check the RED values for each participant
	for _, p := range mockData {
		pId := log.ParticipantId()
		red, ok := redAnalyzer.ResultsByParticipant[pId][p.filePath]
		if !ok {
			t.Errorf("No RED value found for participant %s", pId)
			continue
		}

		if red < p.expectedRED || red > p.expectedRED+0.00001 {
			t.Errorf("Incorrect RED for file %s: got %v, want %v", p.filePath, red, p.expectedRED)
		}
	}
}
