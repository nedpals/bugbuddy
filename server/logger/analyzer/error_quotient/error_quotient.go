package errorquotient

import (
	"fmt"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/nedpals/bugbuddy/server/logger"
	"github.com/nedpals/bugbuddy/server/logger/analyzer"
	"github.com/sergi/go-diff/diffmatchpatch"
)

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
	// open the files first
	content1, err := log.OpenVersionedFile(filepath, version1.FileVersion)
	if err != nil {
		return 0, "", err
	}
	content2, err := log.OpenVersionedFile(filepath, version2.FileVersion)
	if err != nil {
		return 0, "", err
	}

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
	compilationEvents map[string][]CompilationEvent
}

type Analyzer struct {
	LoggerLoaders       []analyzer.LoggerLoader
	ErrorsByParticipant map[string]ErrorQuotientAnalysisResult

	// ResultsByParticipant is a map of participantId to the error quotient
	// map[participantId]map[filePath]errorQuotient
	ResultsByParticipant map[string]map[string]float64
}

func (e *Analyzer) Load(loaders []analyzer.LoggerLoader) error {
	e.LoggerLoaders = loaders
	return nil
}

func (e *Analyzer) Analyze() error {
	for _, loader := range e.LoggerLoaders {
		// Read the log file in a goroutine
		log, err := loader()
		if err != nil {
			continue
		}

		// map[participantId]map[filePath][]logEntry
		logEntries := map[string]map[string][]logger.LogEntry{}

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

			participantId := entry.ParticipantId
			if _, ok := logEntries[participantId]; !ok {
				logEntries[participantId] = map[string][]logger.LogEntry{}
			}

			filePath := entry.FilePath
			if _, ok := logEntries[participantId][filePath]; !ok {
				logEntries[participantId][filePath] = []logger.LogEntry{}
			}

			logEntries[participantId][filePath] = append(logEntries[participantId][filePath], entry)
		}

		for participantId, logEntries := range logEntries {
			// map[filePath][]CompilationEvent
			compilationEvents := map[string][]CompilationEvent{}
			fileNames := []string{}

			for filePath, entries := range logEntries {
				if _, ok := compilationEvents[filePath]; !ok {
					// find the closest file name first before adding the compilation event
					if found := fuzzy.RankFindFold(filePath, fileNames); len(found) != 0 {
						fmt.Printf("error_quotient: Merging %s into %s\n", filePath, found[0].Target)
						filePath = found[0].Target
					} else {
						fileNames = append(fileNames, filePath)
						compilationEvents[filePath] = []CompilationEvent{}
					}
				}

				for i := 0; i < len(entries)-1; i++ {
					entry1 := entries[i]
					entry2 := entries[i+1]

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

					compilationEvents[filePath] = append(compilationEvents[filePath], compilationEvent)
				}
			}

			if e.ErrorsByParticipant == nil {
				e.ErrorsByParticipant = map[string]ErrorQuotientAnalysisResult{}
			}

			// store the compilation events
			e.ErrorsByParticipant[participantId] = ErrorQuotientAnalysisResult{
				compilationEvents: compilationEvents,
			}
		}
	}

	for participantId, result := range e.ErrorsByParticipant {
		if e.ResultsByParticipant == nil {
			e.ResultsByParticipant = map[string]map[string]float64{}
		}

		if _, ok := e.ResultsByParticipant[participantId]; !ok {
			e.ResultsByParticipant[participantId] = map[string]float64{}
		}

		for filePath, events := range result.compilationEvents {
			e.ResultsByParticipant[participantId][filePath] = calculateEQ(events)
		}
	}

	return nil
}
