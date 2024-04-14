package analyzer

import (
	"github.com/nedpals/bugbuddy/server/logger"
)

// KVWriter is an interface for writing key-value pairs
type KVWriter interface {
	Write(name string, pid string, file string, value interface{})
}

type DefaultKVWriter map[string]map[string]map[string]any

func (d DefaultKVWriter) Write(name string, pid string, key string, value interface{}) {
	if _, ok := d[name]; !ok {
		d[name] = make(map[string]map[string]any)
	}

	if _, ok := d[name][pid]; !ok {
		d[name][pid] = make(map[string]any)
	}

	d[name][pid][key] = value
}

func NewDefaultKV() DefaultKVWriter {
	return make(DefaultKVWriter)
}

// LoggerLoader is a function that returns a logger and an error
type LoggerLoader func() (*logger.Logger, error)

// NewLoaderFromPaths creates a new logger loader from a list of log file paths
func NewLoaderFromPaths(paths ...string) []LoggerLoader {
	loaders := make([]LoggerLoader, len(paths))
	for i, path := range paths {
		loaders[i] = func() (*logger.Logger, error) {
			return logger.NewLoggerFromPath(path)
		}
	}

	return loaders
}

// LoadFromExistingLogger creates a new logger loader from an existing logger
func LoadFromExistingLogger(lg *logger.Logger) LoggerLoader {
	return func() (*logger.Logger, error) {
		return lg, nil
	}
}

// LoggerAnalyzer is an interface for analyzing log files
type LoggerAnalyzer interface {
	Analyze(writer KVWriter, loader ...LoggerLoader) error
}

func New[T LoggerAnalyzer]() LoggerAnalyzer {
	instances := make([]T, 1)
	return instances[0]
}
