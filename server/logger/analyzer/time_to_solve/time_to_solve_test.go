package timetosolve_test

import (
	"testing"
	"time"

	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/logger/analyzer"
	timetosolve "github.com/nedpals/bugbuddy/server/logger/analyzer/time_to_solve"
)

func TestTTSAnalyzer(t *testing.T) {
	// Setup logger
	log := logger.NewMemoryLoggerPanic()

	// Mock error and success events to simulate a participant's attempt to solve a problem set
	mockEvents := []struct {
		errorCode   int
		filePath    string
		fileVersion int
		time        time.Time
	}{
		{errorCode: 1, filePath: "/test/problem.go", fileVersion: 1, time: time.Now().Add(-1 * time.Hour)},
		{errorCode: 1, filePath: "/test/problem.go", fileVersion: 2, time: time.Now().Add(-30 * time.Minute)},
		{errorCode: 0, filePath: "/test/problem.go", fileVersion: 3, time: time.Now()},
	}

	// Log the events for the participant
	for _, e := range mockEvents {
		logEntry := logger.LogEntry{
			ErrorCode:   e.errorCode,
			FilePath:    e.filePath,
			FileVersion: e.fileVersion,
			CreatedAt:   &logger.NullTime{Time: e.time, Valid: true},
		}
		if err := log.Log(logEntry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	// Initialize the TTS Analyzer
	ttsa := analyzer.New[*timetosolve.Analyzer]()
	logg := analyzer.LoadFromExistingLogger(log)
	kv := analyzer.NewDefaultKV()

	// Analyze to calculate TTS
	if err := ttsa.Analyze(kv, logg); err != nil {
		t.Fatalf("TTS Analyzer failed: %v", err)
	}

	// Check the TTS value for the participant
	expectedTTS := 1 * time.Hour
	tts, ok := kv[timetosolve.KEY][log.ParticipantId()]["/test/problem.go"].(time.Duration)
	if !ok {
		t.Fatalf("No TTS value found for participant")
	}

	if tts.Round(time.Hour) != expectedTTS {
		t.Errorf("Incorrect TTS for participant: got %v, want %v", tts, expectedTTS)
	}
}
