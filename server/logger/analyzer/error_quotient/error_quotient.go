package errorquotient

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/logger/analyzer"
	"github.com/nedpals/bugbuddy/server/logger/analyzer/internal"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const KEY = "error_quotient"

type CompilationEvent struct {
	ErrorType int
	TimeDelta int // This is the 'T' from your description.
	CharDelta int // This is the 'Ch' from your description.
	Location  string
}

func scoreEventPair(event1, event2 CompilationEvent) int {
	score := 0
	// Check if both events have errors.
	if event1.ErrorType != 0 && event2.ErrorType != 0 {
		score += 2 // Both have errors, add 2.

		// Check if both events have the SAME error type.
		if event1.ErrorType == event2.ErrorType {
			score += 3 // Same error type, add 3.

			// Check if both events have errors at the SAME location.
			if event1.Location == event2.Location {
				score += 3 // Same location, add 3.
			}
		}
	} else if event1.ErrorType != 0 || event2.ErrorType != 0 {
		// If ONLY ONE of the events is an error, it seems you want to add 1 to the score.
		// However, if you're consistently getting 1 for pairs that should possibly score higher,
		// it might be worth revisiting how ErrorType and Location are being determined and compared.
		// As per your description, if they always return 1, it indicates they never match in type and location for scoring 8.
		// For this rule, ensure ErrorType and Location accurately reflect the intended comparisons.
		score += 1 // Only one event is an error, add 1.
	}
	return score
}

func normalizeScore(score int) float64 {
	return float64(score) / 9.0
}

func calculateEQ(events []CompilationEvent) float64 {
	var totalScore float64
	pairCount := len(events) - 1

	if pairCount <= 0 {
		return 0
	}

	for i := 0; i < pairCount; i++ {
		pairScore := scoreEventPair(events[i], events[i+1])
		normalizedScore := normalizeScore(pairScore)
		totalScore += normalizedScore
	}

	eq := totalScore / float64(pairCount)
	return eq
}

// ErrorTypeConversion would convert the ErrorCode to the ErrType used in the EQ calculation.
// For the purpose of this example, assume that any ErrorCode != 0 is an error.
func ErrorTypeConversion(errorCode int) int {
	if errorCode != 0 {
		return 1 // Simplified: 1 represents an error for demonstration purposes.
	}
	return 0
}

// Function to calculate CharDelta by comparing two versions of the same file
func CalculateCharDeltaAndLocation(log *logger.Logger, filepath string, version1, version2 logger.LogEntry) (charDelta int, location string, err error) {
	if version1.FileVersion == 0 && strings.HasPrefix(version1.GeneratedOutput, "# UnknownError") {
		// This is a special case where the file is not found, and the error message is "# UnknownError"
		return 0, "", nil
	}

	// open the files first
	content1, err := log.OpenVersionedFileFromPID(version1.ParticipantId, filepath, version1.FileVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			if version1.ErrorCode != 0 {
				return 0, "", fmt.Errorf("(%s) file not found: %s (version: %d)", version1.ParticipantId, filepath, version1.FileVersion)
			}

			// Just ignore and use the contents from version2
		} else {
			return 0, "", err
		}
	}

	content2, err := log.OpenVersionedFileFromPID(version2.ParticipantId, filepath, version2.FileVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			if version2.ErrorCode != 0 {
				return 0, "", fmt.Errorf("file not found: %s (version: %d)", filepath, version2.FileVersion)
			}

			// Use latest version if the file is not found and error code is 0
			latestVersion, verErr := log.LatestVersionFromFile(filepath)
			if latestVersion == -1 {
				return 0, "", verErr
			}

			newContent2, err := log.OpenVersionedFileFromPID(version2.ParticipantId, filepath, latestVersion)
			if err != nil {
				if err == sql.ErrNoRows {
					return 0, "", fmt.Errorf("(2nd) file not found: %s (version: %d)", filepath, latestVersion)
				}
				return 0, "", err
			}

			// Use the content from version2 if version1 is not found
			if version1.FileVersion == 0 {
				content1 = content2
			}

			content2 = newContent2
		} else {
			return 0, "", err
		}
	}
	err = nil

	// diff the files
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(content1), string(content2), false)

	// calculate the char delta
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete {
			charDelta += len(diff.Text)
		}
	}

	// infer the location
	locations := []string{}

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			// A deletion may indicate fixing an error, include it
			locations = append(locations, fmt.Sprintf("Deleted: %q", diff.Text))
		case diffmatchpatch.DiffInsert:
			// An insertion may indicate adding new code, include it
			locations = append(locations, fmt.Sprintf("Inserted: %q", diff.Text))
			// You can choose to ignore or handle 'DiffEqual' differently, depending on your requirements.
		}
	}

	location = strings.Join(locations, ", ")
	return
}

type ErrorQuotientAnalysisResult struct {
	// errorEntries is a map of error types to the log entries
	// format: map[filePath][]logEntry
	compilationEvents *internal.ResultStore[[]CompilationEvent]
}

type Analyzer struct {
	ErrorsByParticipant map[string]ErrorQuotientAnalysisResult
}

func getLastVersionNumberFromIdx(entries []logger.LogEntry, filePath string, idx int) int {
	for i := idx; i >= 0; i-- {
		if entries[i].FilePath == filePath && entries[i].FileVersion != 0 {
			return entries[i].FileVersion
		}
	}
	return 0
}

func (e *Analyzer) Analyze(writer analyzer.KVWriter, loaders ...analyzer.LoggerLoader) error {
	results := map[string]ErrorQuotientAnalysisResult{}

	for _, loader := range loaders {
		// Read the log file in a goroutine
		log, err := loader()
		if err != nil {
			continue
		}

		// map[participantId]map[filePath][]logEntry
		logEntries := map[string]*internal.ResultStore[[]logger.LogEntry]{}

		// Get all the iter from the logger
		iter, err := log.Entries()
		if err != nil {
			continue
		}

		// Count the number of times each error occurred
		for iter.Next() {
			entry, err := iter.Value()
			if err != nil {
				// break the loop if sql no rows error
				if err.Error() == "sql: no rows in result set" {
					break
				}

				continue
			}

			// Skip if the error message is "file not found". This is not the programmers fault.
			if strings.Contains(entry.ErrorMessage, "error: file not found:") {
				continue
			}

			participantId := entry.ParticipantId
			if _, ok := logEntries[participantId]; !ok {
				logEntries[participantId] = internal.NewResultStore[[]logger.LogEntry]()
			}

			filePath := logEntries[participantId].FilenameNearest(entry.FilePath)
			existing := logEntries[participantId].GetOr(filePath, []logger.LogEntry{})
			logEntries[participantId].Set(filePath, append(existing, entry))
		}

		for participantId, logEntries := range logEntries {
			// map[filePath][]CompilationEvent
			compilationEvents := internal.NewResultStore[[]CompilationEvent]()

			for filePathIdx, entries := range logEntries.Values {
				filePath := logEntries.FilenameNearest(logEntries.Filenames[filePathIdx])

				for i := 0; i < len(entries)-1; i++ {
					entry1 := entries[i]

					// Because the filePath uses the nearestFilename, some entries may have the wrong version number
					// because the original file path is not found in the logEntries.
					if entry1.FileVersion == 0 {
						entry1.FileVersion = getLastVersionNumberFromIdx(entries, filePath, i)
					}

					entry2 := entries[i+1]

					// Same as above, but for the second entry
					if entry2.FileVersion == 0 {
						if entry1.FileVersion != 0 {
							entry2.FileVersion = entry1.FileVersion
						} else {
							entry2.FileVersion = getLastVersionNumberFromIdx(entries, filePath, i+1)
						}
					}

					// Calculate CharDelta between file versions
					charDelta, location, err := CalculateCharDeltaAndLocation(log, filePath, entry1, entry2)
					if err != nil {
						// TODO: replace it with proper error handling
						fmt.Printf("Error calculating char delta: %v\n", err)
						continue
					}

					compilationEvent := CompilationEvent{
						ErrorType: ErrorTypeConversion(entry1.ErrorCode),
						TimeDelta: int(entry2.CreatedAt.Time.Sub(entry1.CreatedAt.Time).Seconds()),
						CharDelta: charDelta,
						Location:  location,
					}

					compilationEvents.Set(
						filePath,
						append(
							compilationEvents.GetOr(filePath, []CompilationEvent{}),
							compilationEvent))
				}
			}

			// store the compilation events
			results[participantId] = ErrorQuotientAnalysisResult{
				compilationEvents: compilationEvents,
			}
		}
	}

	for participantId, result := range results {
		for filePathIdx, events := range result.compilationEvents.Values {
			filePath := result.compilationEvents.Filenames[filePathIdx]
			writer.Write(KEY, participantId, filePath, calculateEQ(events))
		}
	}

	return nil
}
