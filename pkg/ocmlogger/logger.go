package ocmlogger

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// OCMLogger interface should not expose any implementation details like zerolog.Level for example
type OCMLogger interface {
	Err(err error) OCMLogger
	// Extra stores a key-value pair in a map; entry with the same key will be overwritten
	// All simple (non-struct, non-slice, non-map etc.)
	// These values will NOT be send to sentry/glitchtip as `tags`
	Extra(key string, value any) OCMLogger
	AdditionalCallLevelSkips(skip int) OCMLogger
	ClearExtras() OCMLogger
	CaptureSentryEvent(capture bool) OCMLogger
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
}

var _ OCMLogger = &logger{}

type extra map[string]any
type extraCallbacks map[string]func(ctx context.Context) any

//type tags map[string]string

type logger struct {
	ctx                        context.Context
	extra                      extra
	err                        error
	additionalCallLevelSkips   int
	captureSentryEventOverride *bool
	lock                       sync.RWMutex
}

var (
	OCM_LOG_LEVEL_FLAG_NAME = "u"
	OCM_LOG_TRACE           = zerolog.TraceLevel.String()
	OCM_LOG_DEBUG           = zerolog.DebugLevel.String()
	OCM_LOG_INFO            = zerolog.InfoLevel.String()
	OCM_LOG_WARN            = zerolog.WarnLevel.String()
	OCM_LOG_ERROR           = zerolog.ErrorLevel.String()
	OCM_LOG_LEVEL_DEFAULT   = OCM_LOG_WARN

	rootLogger zerolog.Logger // root logger used by our application

	// log.Info("foo") -> OCMLogger.Info -> OCMLogger.log -> OCMLogger.hydrateLog -> log library ...
	// If we don't provide a base offset of 3, it will appear as if all logs are coming from OCMLogger.hydrateLog
	baseCallerSkipLevel = 3

	trimList = []string{"pkg"}

	// The context in go requires key to be `exactly the same` both when setting with context.WithValue and getting with context.Value
	// It makes library being unable to fetch those keys by itself if they are not `string` (even if its underlying type is `string`
	// For example:
	//
	// 		const OpIDKey OperationIDKey = "opID"
	// 		opID := util.NewID()
	// 		ctx = context.WithValue(ctx, OpIDKey, opID)
	//
	// 		opID, ok := ctx.Value(OpIDKey).(string) -- this will work
	// 		opID, ok := ctx.Value("opID").(string) -- this will NOT work
	//
	// We allow you to register callback functions to safely retrieve values from the context. The values returned
	// by those functions will be added to the log as `Extra` fields under the provided key.
	retrieveExtraFromContextCallbacks = make(extraCallbacks)
)

var possibleLogLevels = []string{
	zerolog.LevelTraceValue,
	zerolog.LevelDebugValue,
	zerolog.LevelInfoValue,
	zerolog.LevelWarnValue,
	zerolog.LevelErrorValue,
	zerolog.LevelFatalValue,
	zerolog.LevelPanicValue,
}

var sentryLevelMapping = map[zerolog.Level]sentry.Level{
	zerolog.TraceLevel: sentry.LevelDebug,
	zerolog.DebugLevel: sentry.LevelDebug,
	zerolog.InfoLevel:  sentry.LevelInfo,
	zerolog.WarnLevel:  sentry.LevelWarning,
	zerolog.ErrorLevel: sentry.LevelError,
	zerolog.FatalLevel: sentry.LevelFatal,
}

/**
 * Flag parsing for our logger is a bit different from the rest of our app because
 * we want basic logging to work as early as possible in the application lifecycle.
 * In particular we want logging to work before any of our complex environment
 * initialization so that we can rely on logs for debugging if necessary.
 */
func init() {
	rootLogger = log.Logger
	_ = SetLogLevel(OCM_LOG_LEVEL_DEFAULT)

	// register a callback function, so we can update state when flags are parsed
	flag.Func(OCM_LOG_LEVEL_FLAG_NAME, fmt.Sprintf("Log level, one of: %v", possibleLogLevels), func(s string) error {
		if s == "" { // for some reason its blank sometimes...
			s = OCM_LOG_LEVEL_DEFAULT
		}
		return SetLogLevel(s)
	})

	// CallerMarshalFunc allows customization of global caller marshaling, i.e. .Caller()
	// Used to trim caller file paths, so they look a bit nicer
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		file = strings.ReplaceAll(file, "\\", "/")
		for _, t := range trimList {
			if i := strings.Index(file, t); i > -1 {
				file = file[i:]
				break
			}
		}

		return file + ":" + strconv.Itoa(line)
	}
}

// NewOCMLogger creates a logger and initializes it
// This ensures that each thread will get its own logger
func NewOCMLogger(ctx context.Context) OCMLogger {
	return &logger{
		extra: make(extra),
		ctx:   ctx,
	}
}

// SetLogLevel - update logger state to a new level
func SetLogLevel(level string) error {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(l) // try to keep zerolog global logger in sync even though we dont use it
	rootLogger = rootLogger.Level(l)
	return nil
}

// SetOutput - used for testing
func SetOutput(output io.Writer) {
	rootLogger = rootLogger.Output(output)
}

func RegisterExtraDataCallback(key string, callback func(ctx context.Context) any) {
	retrieveExtraFromContextCallbacks[key] = callback
}

func ClearExtraDataCallbacks() {
	retrieveExtraFromContextCallbacks = make(extraCallbacks)
}

func SetTrimList(trims []string) {
	trimList = trims
}

func TraceEnabled() bool {
	return logLevelEnabled(zerolog.TraceLevel)
}

func DebugEnabled() bool {
	return logLevelEnabled(zerolog.DebugLevel)
}

func InfoEnabled() bool {
	return logLevelEnabled(zerolog.InfoLevel)
}

func WarnEnabled() bool {
	return logLevelEnabled(zerolog.WarnLevel)
}

func ErrorEnabled() bool {
	return logLevelEnabled(zerolog.ErrorLevel)
}

func (l *logger) Err(err error) OCMLogger {
	l.err = err
	return l
}

// AdditionalCallLevelSkips - allows to skip additional frames when logging, useful for wrapping loggers like OcmSdkLogWrapper
func (l *logger) AdditionalCallLevelSkips(skip int) OCMLogger {
	l.additionalCallLevelSkips = skip
	return l
}

func (l *logger) Extra(key string, value any) OCMLogger {
	l.lock.Lock()
	l.extra[key] = value
	l.lock.Unlock()
	return l
}

func (l *logger) ClearExtras() OCMLogger {
	l.extra = make(extra)
	return l
}

func (l *logger) CaptureSentryEvent(capture bool) OCMLogger {
	l.captureSentryEventOverride = &capture
	return l
}

func (l *logger) Info(args ...any) {
	l.log(zerolog.InfoLevel, args...)
}

func (l *logger) Debug(args ...any) {
	l.log(zerolog.DebugLevel, args...)
}

func (l *logger) Trace(args ...any) {
	l.log(zerolog.TraceLevel, args...)
}

func (l *logger) Warning(args ...any) {
	l.log(zerolog.WarnLevel, args...)
}

func (l *logger) Error(args ...any) {
	l.log(zerolog.ErrorLevel, args...)
}

func (l *logger) Fatal(args ...any) {
	l.log(zerolog.FatalLevel, args...)
}

// Note: use the various "Depth" logging functions, so we get the correct file/line number in the logs
func (l *logger) log(level zerolog.Level, args ...any) {

	defer func() {
		l.Err(nil)
		l.ClearExtras()
	}()

	message := ""
	if len(args) > 0 {
		message = args[0].(string)
		args = args[1:]
	}

	if message == "" && l.err != nil {
		message = l.err.Error()
	}

	// by default only capture sentry events for error and fatal levels
	captureSentry := level == zerolog.ErrorLevel || level == zerolog.FatalLevel

	// if caller explicitly overrides the captureSentryEvent, use that instead
	if l.captureSentryEventOverride != nil {
		captureSentry = *l.captureSentryEventOverride
	}

	if captureSentry {
		sentryId := l.tryCaptureSentryEvent(level, message, args...)
		if sentryId != nil {
			l.Extra("SentryEventID", sentryId)
		}
	}

	event := l.hydrateLog(level)

	if event.Enabled() {
		event.Msgf(message, args...)
		event.Discard()
	}

	if level == zerolog.FatalLevel {
		os.Exit(1)
	}
}

func (l *logger) tryCaptureSentryEvent(level zerolog.Level, message string, args ...any) *sentry.EventID {
	event := sentry.NewEvent()
	event.Level = sentryLevelMapping[level]
	event.Message = fmt.Sprintf(message, args...)
	event.Fingerprint = []string{getMD5Hash(event.Message)}
	l.lock.RLock()
	event.Extra = l.extra
	defer l.lock.RUnlock()

	if l.err != nil || level == zerolog.ErrorLevel || level == zerolog.FatalLevel {
		var sentryStack *sentry.Stacktrace
		if l.err != nil {
			// support errors that include a stacktrace, such as github.com/pkg/errors
			sentryStack = sentry.ExtractStacktrace(l.err)
		}
		if sentryStack == nil {
			sentryStack = sentry.NewStacktrace()

			// remove the frames that are not relevant to the caller
			// last frame of the stacktrace should be the line where the user called ulog.Error(...)
			framesToKeep := len(sentryStack.Frames) - baseCallerSkipLevel - l.additionalCallLevelSkips
			sentryStack.Frames = sentryStack.Frames[:framesToKeep]
		}

		// Add an exception to the event. Note that we use the log message as the `Type` of the error
		// because that is what Sentry uses as the title for the issue, and types of errors in Go
		// are usually not very useful.
		event.Exception = []sentry.Exception{
			{
				Type:       event.Message,
				Value:      event.Message,
				Stacktrace: sentryStack,
			},
		}
	}

	sentryHub := sentry.GetHubFromContext(l.ctx)
	if sentryHub == nil {
		sentryHub = sentry.CurrentHub()
	}

	if sentryHub == nil {
		return nil
	}
	return sentryHub.CaptureEvent(event)
}

func (l *logger) hydrateLog(level zerolog.Level) *zerolog.Event {
	event := rootLogger.WithLevel(level).
		Caller(baseCallerSkipLevel + l.additionalCallLevelSkips).
		Err(l.err)

	for k, callback := range retrieveExtraFromContextCallbacks {
		if callback != nil {
			v := callback(l.ctx)
			l.Extra(k, v)
		}
	}

	if len(l.extra) > 0 {
		dict := zerolog.Dict()
		l.lock.RLock()
		for k, v := range l.extra {
			switch typ := v.(type) {
			case string:
				dict.Str(k, typ)
			case bool:
				dict.Bool(k, typ)
			case int:
				dict.Int(k, typ)
			case int8:
				dict.Int8(k, typ)
			case int16:
				dict.Int16(k, typ)
			case int32:
				dict.Int32(k, typ)
			case int64:
				dict.Int64(k, typ)
			case float32:
				dict.Float32(k, typ)
			case float64:
				dict.Float64(k, typ)
			default:
				// Note: reflection is expensive, but we need it to add structs like `http.Request` or `http.Response` etc.
				dict.Any(k, v)
			}
		}
		l.lock.RUnlock()
		event.Dict("Extra", dict)
	}
	return event
}

func logLevelEnabled(callLevel zerolog.Level) bool {
	configLevel := rootLogger.GetLevel()
	return configLevel != zerolog.Disabled && configLevel != zerolog.NoLevel && configLevel <= callLevel
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
