package timetosolve

import (
	"time"

	"github.com/nedpals/bugbuddy/server/logger/analyzer"
	"github.com/nedpals/bugbuddy/server/logger/analyzer/internal"
)

const KEY = "time_to_solve"

type Analyzer struct{}

func (t *Analyzer) Analyze(writer analyzer.KVWriter, loaders ...analyzer.LoggerLoader) error {
	for _, loader := range loaders {
		log, err := loader()
		if err != nil {
			return err
		}

		iter, err := log.Entries()
		if err != nil {
			return err
		}

		// map[participantId]map[filePath]time.Time
		startTimes := map[string]*internal.ResultStore[time.Time]{}

		for iter.Next() {
			entry, err := iter.Value()
			if err != nil {
				continue
			}

			filePath := entry.FilePath

			if _, ok := startTimes[entry.ParticipantId]; !ok {
				startTimes[entry.ParticipantId] = internal.NewResultStore[time.Time]()
			}

			if !startTimes[entry.ParticipantId].Has(filePath) {
				startTime := startTimes[entry.ParticipantId].GetOr(filePath, time.Time{})

				// if startTime is zero or startTime is greater than entry.CreatedAt.Time
				// set the startTime to entry.CreatedAt.Time
				if startTime.IsZero() || startTime.Compare(entry.CreatedAt.Time) > 0 {
					startTimes[entry.ParticipantId].Set(filePath, entry.CreatedAt.Time)
				}
			}

			// If this entry represents a successful compilation, update the TTS
			if entry.ErrorCode == 0 {
				startTime := startTimes[entry.ParticipantId].Get(filePath)
				writer.Write(KEY, entry.ParticipantId, filePath, entry.CreatedAt.Time.Sub(startTime))
			}
		}
	}

	return nil
}
