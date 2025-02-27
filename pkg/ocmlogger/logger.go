package ocmlogger

import (
	"bytes"
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
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// OCMLogger interface should not expose any implementation details like zerolog.Level for example
type OCMLogger interface {
	// Contextual provides an interface with standard contextual logging libraries.
	// It is required for providing additional key/value pairs like the old Err and Extra functions.
	Contextual() ContextualLogger

	AdditionalCallLevelSkips(skip int) OCMLogger
	CaptureSentryEvent(capture bool) OCMLogger
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
}

var _ OCMLogger = &logger{}

type extraCallbacks map[string]func(ctx context.Context) any

type logger struct {
	ctx                      context.Context
	additionalCallLevelSkips atomic.Int32

	captureSentrySet           atomic.Bool
	captureSentryEventOverride atomic.Bool

	// Thread Safety Note: We use a read-write lock to protect the `extra` map so that concurrent writes
	// dont cause a panic, however, the logger is not fundamentally designed to be used concurrently as a
	// communication channel. Each thread/goroutine should have its own internal logger instance. We make
	// no concurrency guarantees other than "it wont blow up."
	lock sync.RWMutex
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

	// log.Info("foo") -> OCMLogger.Info -> OCMLogger.log -> OCMLogger.createLogEvent -> log library ...
	// If we don't provide a base offset of 3, it will appear as if all logs are coming from OCMLogger.createLogEvent
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
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

// NewOCMLogger creates a logger and initializes it
// This ensures that each thread will get its own logger
func NewOCMLogger(ctx context.Context) OCMLogger {
	return &logger{
		ctx:                      ctx,
		additionalCallLevelSkips: atomic.Int32{},
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
// Whenever used, please be sure your io.Writer is threadsafe or you will end up with data races.
// If you are testing, WrapUnsafeWriterWithLocks is an easy function to use to ensure this.
func SetOutput(output io.Writer) {
	rootLogger = rootLogger.Output(output)
}

type ThreadSafeBytesBuffer interface {
	io.Writer
	io.Reader
	String() string
}

// WrapUnsafeWriterWithLocks wraps any io.Writer with a lock during .Write to ensure guaranteed ordering.
// Note, this does NOT mean that writes will not be interleaved since the library can choose how many bytes to write at once.
func WrapUnsafeWriterWithLocks(writer *bytes.Buffer) ThreadSafeBytesBuffer {
	return &threadSafeWriter{
		delegate: writer,
	}
}

type threadSafeWriter struct {
	lock     sync.Mutex
	delegate *bytes.Buffer
}

var _ ThreadSafeBytesBuffer = &threadSafeWriter{}

func (t *threadSafeWriter) Write(p []byte) (n int, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.delegate.Write(p)
}

func (t *threadSafeWriter) Read(p []byte) (n int, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.delegate.Read(p)
}

func (t *threadSafeWriter) String() string {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.delegate.String()
}

var _ io.Writer = &threadSafeWriter{}

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

// AdditionalCallLevelSkips - allows to skip additional frames when logging, useful for wrapping loggers like OcmSdkLogWrapper
func (l *logger) AdditionalCallLevelSkips(skip int) OCMLogger {
	l.additionalCallLevelSkips.Store(int32(skip))
	return l
}

func (l *logger) CaptureSentryEvent(capture bool) OCMLogger {
	l.captureSentryEventOverride.Store(capture)
	l.captureSentrySet.Store(true)
	return l
}

// returns
func (l *logger) getCaptureSentryEvent() (captureSentry bool, overridden bool) {
	return l.captureSentryEventOverride.Load(), l.captureSentrySet.Load()
}

func (l *logger) Info(args ...any) {
	l.legacyLog(zerolog.InfoLevel, args)
}

func (l *logger) Debug(args ...any) {
	l.legacyLog(zerolog.DebugLevel, args)
}

func (l *logger) Trace(args ...any) {
	l.legacyLog(zerolog.TraceLevel, args)
}

func (l *logger) Warning(args ...any) {
	l.legacyLog(zerolog.WarnLevel, args)
}

func (l *logger) Fatal(args ...any) {
	l.legacyLog(zerolog.FatalLevel, args)
}

func (l *logger) Error(args ...any) {
	l.legacyLog(zerolog.ErrorLevel, args)
}

func (l *logger) legacyLog(level zerolog.Level, args []any) {
	if len(args) == 0 {
		l.log(level, "", nil, nil)
		return
	}

	messageString, isString := args[0].(string)
	if !isString { // stringify if it's not actually a string
		messageString = fmt.Sprintf("%v", args[0])
	}

	if len(args) == 1 {
		l.log(level, messageString, nil, nil)
		return
	}

	l.log(level, fmt.Sprintf(messageString, args[1:]...), nil, nil)
}

// Note: use the various "Depth" logging functions, so we get the correct file/line number in the logs
func (l *logger) log(level zerolog.Level, message string, err error, keysAndValues []interface{}) {
	if message == "" && err != nil {
		message = err.Error()
	}

	// by default only capture sentry events for error and fatal levels
	captureSentry := level == zerolog.ErrorLevel || level == zerolog.FatalLevel

	// if caller explicitly overrides the captureSentryEvent, use that instead
	if captureSentryOverride, overridden := l.getCaptureSentryEvent(); overridden {
		captureSentry = captureSentryOverride
	}

	// make sure we have all the extras from the context before trying to capture the sentry event
	keysAndValues = append(keysAndValues, extrasFromContext(l.ctx)...)

	if captureSentry {
		sentryId := l.tryCaptureSentryEvent(level, message, err, keysAndValues)
		if sentryId != nil {
			keysAndValues = append(keysAndValues, "SentryEventID", sentryId)
		}
	}

	event := l.createLogEvent(level, err, keysAndValues)
	// once an zerolog event is created, it is imperative that we call .Msg on it so that the event will be returned to the pool.
	// if we don't do this, we leak from the pool which continues to grow.
	// After calling .Msg, it is imperative that we do not call any additional methods on event because the pool is reused
	// and this can cause data races in the library.
	// if we ever move off zerolog, this likely requires consideration.
	event.Msg(message)

	if level == zerolog.FatalLevel {
		os.Exit(1)
	}
}

func (l *logger) tryCaptureSentryEvent(level zerolog.Level, message string, err error, keysAndValues []interface{}) *sentry.EventID {
	event := sentry.NewEvent()
	event.Level = sentryLevelMapping[level]
	event.Message = message
	event.Fingerprint = []string{getMD5Hash(event.Message)}
	event.Extra = contextToLegacyExtra(keysAndValues)

	if err != nil || level == zerolog.ErrorLevel || level == zerolog.FatalLevel {
		var sentryStack *sentry.Stacktrace
		if err != nil {
			// support errors that include a stacktrace, such as github.com/pkg/errors
			sentryStack = sentry.ExtractStacktrace(err)
		}
		if sentryStack == nil {
			sentryStack = sentry.NewStacktrace()

			// remove the frames that are not relevant to the caller
			// last frame of the stacktrace should be the line where the user called ulog.Error(...)
			framesToKeep := len(sentryStack.Frames) - baseCallerSkipLevel - int(l.additionalCallLevelSkips.Load())
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

func extrasFromContext(ctx context.Context) []interface{} {
	ret := []interface{}{}
	for k, callback := range retrieveExtraFromContextCallbacks {
		if callback != nil {
			v := callback(ctx)
			ret = append(ret, k, v)
		}
	}
	return ret
}

const (
	// missingValue matches the klog missing value marker
	missingValue = "(MISSING)"
)

func contextToLegacyExtra(keysAndValues []interface{}) map[string]interface{} {
	if len(keysAndValues) == 0 {
		return nil
	}

	ret := map[string]interface{}{}
	currKey := "" // tracked to handle mismatch in len
	for i, curr := range keysAndValues {
		isKey := i%2 == 0
		if isKey {
			var ok bool
			currKey, ok = curr.(string)
			switch {
			case !ok:
				currKey = fmt.Sprintf("(NOT_A_STRING[%d])", i)
			case len(currKey) == 0:
				currKey = fmt.Sprintf("(MISSING_KEY[%d])", i)
			}
			continue
		}

		ret[currKey] = curr

		// reset
		currKey = ""
	}

	if len(currKey) > 0 {
		ret[currKey] = missingValue
	}

	return ret
}

func (l *logger) createLogEvent(level zerolog.Level, err error, extraKeysAndValues []interface{}) *zerolog.Event {
	event := rootLogger.WithLevel(level).
		Caller(baseCallerSkipLevel + int(l.additionalCallLevelSkips.Load())).
		Err(err)

	if len(extraKeysAndValues) > 0 {
		// this extra nesting is required for serialization equality with old serializations.
		// TODO 1. choose new serialization and start producing it
		// TODO 2. update all consuming code to handle a new serialization format
		// TODO 3. remove this extra nesting
		event = event.Fields([]interface{}{"Extra", contextToLegacyExtra(extraKeysAndValues)})
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

func (l *logger) Contextual() ContextualLogger {
	return &contextualWrapper{delegate: l}
}
