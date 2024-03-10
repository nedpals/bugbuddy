package analyzer

import "github.com/nedpals/bugbuddy/server/logger"

// LoggerLoader is a function that returns a logger and an error
type LoggerLoader func() (*logger.Logger, error)

// LoadFromExistingLogger creates a new logger loader from an existing logger
func LoadFromExistingLogger(lg *logger.Logger) LoggerLoader {
	return func() (*logger.Logger, error) {
		return lg, nil
	}
}

// LoggerAnalyzer is an interface for analyzing log files
type LoggerAnalyzer[A any] interface {
	Load(loader []LoggerLoader) error
	Analyze() error
	*A
}

// NewAnalyzerFromPaths creates a new analyzer from a list of log file paths
func NewAnalyzerFromPaths[T any, PT LoggerAnalyzer[T]](paths ...string) PT {
	loaders := make([]LoggerLoader, len(paths))
	for i, path := range paths {
		loaders[i] = func() (*logger.Logger, error) {
			return logger.NewLoggerFromPath(path)
		}
	}

	return New[T, PT](loaders...)
}

// New creates a new analyzer from a list of log loaders
func New[T any, PT LoggerAnalyzer[T]](loader ...LoggerLoader) PT {
	instances := make([]T, 1)
	an := PT(&instances[0])
	an.Load(loader)
	return an
}

// Re-export the analyzers
type ErrorQuotient = errorQuotientAnalyzer
