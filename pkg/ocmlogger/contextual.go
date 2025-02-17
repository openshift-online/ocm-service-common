package ocmlogger

import (
	"github.com/rs/zerolog"
)

// CompatibleContextualLogger is the interface used in this package for logging, so that any backend
// can be plugged in. It is a subset of the github.com/go-logr/logr and the k8s.io contextual logging interface.
type CompatibleContextualLogger interface {
	// Info logs a non-error message with the given key/value pairs as context.
	//
	// The msg argument should be used to add some constant description to the log
	// line.  The key/value pairs can then be used to add additional variable
	// information.  The key/value pairs must alternate string keys and arbitrary
	// values.
	Info(msg string, keysAndValues ...interface{})

	// Error logs an error, with the given message and key/value pairs as context.
	// It functions similarly to Info, but may have unique behavior, and should be
	// preferred for logging errors (see the package documentations for more
	// information). The log message will always be emitted, regardless of
	// verbosity level.
	//
	// The msg argument should be used to add context to any underlying error,
	// while the err argument should be used to attach the actual error that
	// triggered this log line, if present. The err parameter is optional
	// and nil may be passed instead of an error instance.
	Error(err error, msg string, keysAndValues ...interface{})
}

// ContextualLogger is a CompatibleContextualLogger plus the methods that are not part of the standard interface,
// but are needed to easily close .
type ContextualLogger interface {
	CompatibleContextualLogger

	Debug(msg string, keysAndValues ...interface{})
	Trace(msg string, keysAndValues ...interface{})
	InfoWithError(err error, msg string, keysAndValues ...interface{})
	Warning(msg string, keysAndValues ...interface{})
	WarningWithError(err error, msg string, keysAndValues ...interface{})
	Fatal(err error, msg string, keysAndValues ...interface{})
}

type contextualWrapper struct {
	delegate *logger
}

var _ ContextualLogger = &contextualWrapper{}

func (c *contextualWrapper) Debug(msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.DebugLevel, msg, nil, keysAndValues)
}

func (c *contextualWrapper) Trace(msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.TraceLevel, msg, nil, keysAndValues)
}

func (c *contextualWrapper) Info(msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.InfoLevel, msg, nil, keysAndValues)
}

func (c *contextualWrapper) InfoWithError(err error, msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.InfoLevel, msg, err, keysAndValues)
}

func (c *contextualWrapper) Warning(msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.WarnLevel, msg, nil, keysAndValues)
}

func (c *contextualWrapper) WarningWithError(err error, msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.WarnLevel, msg, err, keysAndValues)
}

func (c *contextualWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.ErrorLevel, msg, err, keysAndValues)
}

func (c *contextualWrapper) Fatal(err error, msg string, keysAndValues ...interface{}) {
	c.delegate.log(zerolog.FatalLevel, msg, err, keysAndValues)
}
