package gonest

import (
	"fmt"
	"log"
	"os"
)

// Logger defines the logging interface used throughout the framework.
type Logger interface {
	Log(format string, args ...any)
	Error(format string, args ...any)
	Warn(format string, args ...any)
	Debug(format string, args ...any)
}

// DefaultLogger is the built-in logger backed by the standard library.
type DefaultLogger struct {
	logger *log.Logger
	debug  bool
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "[GoNest] ", log.LstdFlags),
	}
}

func NewDefaultLoggerWithDebug() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "[GoNest] ", log.LstdFlags),
		debug:  true,
	}
}

func (l *DefaultLogger) Log(format string, args ...any) {
	l.logger.Printf("LOG   "+format, args...)
}

func (l *DefaultLogger) Error(format string, args ...any) {
	l.logger.Printf("ERROR "+format, args...)
}

func (l *DefaultLogger) Warn(format string, args ...any) {
	l.logger.Printf("WARN  "+format, args...)
}

func (l *DefaultLogger) Debug(format string, args ...any) {
	if l.debug {
		l.logger.Printf("DEBUG "+format, args...)
	}
}

// NopLogger discards all log output.
type NopLogger struct{}

func (NopLogger) Log(string, ...any)   {}
func (NopLogger) Error(string, ...any) {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Debug(string, ...any) {}

// Sprintf is a convenience alias for fmt.Sprintf.
var Sprintf = fmt.Sprintf
