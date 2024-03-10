package timetosolve

import (
	"time"

	"github.com/nedpals/bugbuddy/server/logger/analyzer"
)

type Analyzer struct {
	LoggerLoader []analyzer.LoggerLoader
	// ResultsByParticipant is a map of participant IDs to their respective Time-to-Solve for each problem set.
	// format: map[participantId]map[filePath]time.Duration
	ResultsByParticipant map[string]map[string]time.Duration
}

func (t *Analyzer) Load(loader []analyzer.LoggerLoader) error {
	t.LoggerLoader = loader
	return nil
}

func (t *Analyzer) Analyze() error {
	for _, loader := range t.LoggerLoader {
		log, err := loader()
		if err != nil {
			return err
		}

		iter, err := log.Entries()
		if err != nil {
			return err
		}

		// map[participantId]map[filePath]time.Duration
		ttsResults := map[string]map[string]time.Duration{}

		// map[participantId]map[filePath]time.Time
		startTimes := map[string]map[string]time.Time{}

		for iter.Next() {
			entry, err := iter.Value()
			if err != nil {
				continue
			}

			// Initialize the participantId and filePath maps if they haven't been already
			if _, ok := ttsResults[entry.ParticipantId]; !ok {
				ttsResults[entry.ParticipantId] = make(map[string]time.Duration)
			}

			if _, ok := ttsResults[entry.ParticipantId][entry.FilePath]; !ok {
				// For simplicity, assume that the first entry is the start time
				// This may need to be adjusted depending on how you define the start of a problem set
				ttsResults[entry.ParticipantId][entry.FilePath] = time.Duration(0)
			}

			if _, ok := startTimes[entry.ParticipantId]; !ok {
				startTimes[entry.ParticipantId] = make(map[string]time.Time)
				startTimes[entry.ParticipantId][entry.FilePath] = entry.CreatedAt.Time
			}

			// If this entry represents a successful compilation, update the TTS
			if entry.ErrorCode == 0 {
				startTime := startTimes[entry.ParticipantId][entry.FilePath]
				ttsResults[entry.ParticipantId][entry.FilePath] = entry.CreatedAt.Time.Sub(startTime)
			}
		}

		t.ResultsByParticipant = ttsResults
	}

	return nil
}
