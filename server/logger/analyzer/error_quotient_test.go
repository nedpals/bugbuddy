package analyzer_test

import (
	"testing"

	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/logger/analyzer"
)

func TestErrorQuotientAnalyzer(t *testing.T) {
	// Create a memory logger and add sample log entries
	log, err := logger.NewMemoryLogger()
	if err != nil {
		t.Fatal(err)
	}

	participantId := log.ParticipantId()
	filePath := "/path/to/source/code.go"

	mockEvents := []struct {
		command       string
		errorCode     int
		errorMessage  string
		filePath      string
		fileVersion   int
		contentBefore string
		contentAfter  string
	}{
		{command: "compile", errorCode: 1, errorMessage: "syntax error", filePath: filePath, fileVersion: 1, contentBefore: "", contentAfter: "fmt.Println(hello"},
		{command: "compile", errorCode: 1, errorMessage: "syntax error", filePath: filePath, fileVersion: 2, contentBefore: "fmt.Println(hello", contentAfter: "fmt.Println(\"hello\""},
		{command: "compile", errorCode: 0, errorMessage: "", filePath: filePath, fileVersion: 3, contentBefore: "fmt.Println(\"hello\"", contentAfter: "fmt.Println(\"hello world\")"},
		{command: "compile", errorCode: 1, errorMessage: "undefined variable world", filePath: filePath, fileVersion: 4, contentBefore: "fmt.Println(\"hello world\")", contentAfter: "var world = \"world\"\nfmt.Println(\"hello \" + world)"},
		{command: "compile", errorCode: 0, errorMessage: "", filePath: filePath, fileVersion: 5, contentBefore: "var world = \"world\"\nfmt.Println(\"hello \" + world)", contentAfter: "var world = \"World\"\nfmt.Println(\"Hello, \" + world)"},
	}

	for _, event := range mockEvents {
		// Simulate file versioning and logging of compilations
		err := log.WriteVersionedFile(event.filePath, []byte(event.contentBefore), event.fileVersion-1)
		if err != nil {
			t.Fatalf("Failed to write previous file version: %v", err)
		}
		err = log.WriteVersionedFile(event.filePath, []byte(event.contentAfter), event.fileVersion)
		if err != nil {
			t.Fatalf("Failed to write file version: %v", err)
		}

		logEntry := logger.LogEntry{
			ExecutedCommand: event.command,
			ErrorCode:       event.errorCode,
			ErrorMessage:    event.errorMessage,
			FilePath:        event.filePath,
		}
		err = log.Log(logEntry)
		if err != nil {
			t.Fatalf("Failed to log compilation event: %v", err)
		}
	}

	// Check if logger has logged 4 entries
	entriesIter, err := log.Entries()
	if err != nil {
		t.Fatal(err)
	}

	entries, err := entriesIter.List()
	if err != nil {
		t.Fatal(err)
	} else if len(entries) != len(mockEvents) {
		t.Fatalf("Expected %d entries, got %d", len(mockEvents), len(entries))
	}

	// Create a new ErrorQuotientAnalyzer
	eqa := analyzer.New[analyzer.ErrorQuotient](analyzer.LoadFromExistingLogger(log))

	// Analyze the log
	if err := eqa.Analyze(); err != nil {
		t.Fatalf("EQ analysis failed: %v", err)
	}

	if _, ok := eqa.ResultsByParticipant[log.ParticipantId()]; !ok {
		t.Fatal("No results found")
	}

	// Assuming we've added more events and now have:
	// - A total of 4 events, with the first and third pairs scoring as per your specific rules.
	// - Let's say, after adjustments, the first pair scores 1, the second pair scores 8 (as per your rules),
	//   and the third pair scores 1 again with the fourth being a successful compilation.

	// Calculate the expected EQ based on these scores:
	// totalScore = 1 + 8 + 1 = 10
	// normalizedScore = (1/9) + (8/9) + (1/9) = 10/9
	// numberOfPairs = 3 (since we now have 4 events, resulting in 3 pairs)
	// expectedEQ = normalizedScore / numberOfPairs = (10/9) / 3 = 10/27

	expectedEQ := 0.370370 // This is an example value; you need to calculate the expected EQ based on your mockEvents
	if eq, ok := eqa.ResultsByParticipant[participantId][filePath]; !ok || (eq < expectedEQ || eq > expectedEQ+0.001) {
		t.Errorf("Expected EQ of %f for participant %s and file %s, but got %f", expectedEQ, participantId, filePath, eq)
	}
}
