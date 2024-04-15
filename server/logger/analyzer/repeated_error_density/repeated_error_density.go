package repeatederrordensity

import (
	"fmt"
	"os"
	"strings"

	"github.com/nedpals/bugbuddy/server/logger/analyzer"
)

const KEY = "repeated_error_density"

// ErrorEvent represents a compilation attempt and whether it was an error.
type ErrorEvent struct {
	IsError   bool
	ErrorType string // This can be an error code or message to identify error types.
}

type Analyzer struct{}

func (e *Analyzer) Analyze(writer analyzer.KVWriter, loaders ...analyzer.LoggerLoader) error {
	for _, loader := range loaders {
		log, err := loader()
		if err != nil {
			continue
		}

		iter, err := log.Entries()
		if err != nil {
			continue
		}

		// map[participantId]map[filePath][]ErrorEvent
		errorEvents := map[string]map[string][]ErrorEvent{}

		for iter.Next() {
			entry, err := iter.Value()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			// skip if the error message is "file not found". This is not the programmers fault.
			if strings.Contains(entry.ErrorMessage, "error: file not found:") {
				continue
			}

			if _, ok := errorEvents[entry.ParticipantId]; !ok {
				errorEvents[entry.ParticipantId] = map[string][]ErrorEvent{}
			}

			if _, ok := errorEvents[entry.ParticipantId][entry.FilePath]; !ok {
				errorEvents[entry.ParticipantId][entry.FilePath] = []ErrorEvent{}
			}

			errorType := entry.ErrorType
			if entry.ErrorCode != 0 && len(errorType) == 0 && strings.Contains(entry.GeneratedOutput, "# UnknownError") {
				errorType = "UnknownError"
			}

			errorEvents[entry.ParticipantId][entry.FilePath] = append(errorEvents[entry.ParticipantId][entry.FilePath], ErrorEvent{
				IsError:   entry.ErrorCode != 0,
				ErrorType: errorType,
			})
		}

		for participantId, eventsByFilepath := range errorEvents {
			for filePath, events := range eventsByFilepath {
				currentErrorType := ""
				repeatedCount := 0
				red := 0.0

				for _, event := range events {
					if event.IsError && event.ErrorType == currentErrorType {
						// Increase the count if the current error is the same as the last one.
						repeatedCount++
						continue
					} else if !event.IsError {
						currentErrorType = ""
					} else if event.ErrorType != currentErrorType {
						currentErrorType = event.ErrorType
					}

					// Fallback if not the same or not an error.

					// Calculate RED for the previous error string and reset the count.
					if repeatedCount > 0 {
						red += float64(repeatedCount*repeatedCount) / float64(repeatedCount+1)
					}

					// Reset the count.
					repeatedCount = 0
				}

				if repeatedCount > 0 {
					red += float64(repeatedCount*repeatedCount) / float64(repeatedCount+1)
				}

				writer.Write(KEY, participantId, filePath, red)
			}
		}
	}

	return nil
}
