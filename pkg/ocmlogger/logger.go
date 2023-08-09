package ocmlogger

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// OCMLogger interface should not expose any implementation details like zerolog.Level for example
type OCMLogger interface {
	Err(err error) OCMLogger
	// Extra adds a key-value pair to "extra" hash, where `value` has a `built-in` type
	Extra(key string, value any) OCMLogger
	// ExtraDeepReflect creates a `key` hash, and adds to it a `value.field-name`-`value.field-value` pairs,
	// where `value` has a `struct/slice/map` type and requires a use of `reflection`
	ExtraDeepReflect(key string, values any) OCMLogger
	AdditionalCallLevelSkips(skip int) OCMLogger
	ClearExtras() OCMLogger
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
}

var _ OCMLogger = &logger{}

type extra map[string]any
type tags map[string]string
type dict struct {
	name   string
	values any
}

type logger struct {
	ctx                      context.Context
	dict                     []dict
	extra                    extra
	err                      error
	additionalCallLevelSkips int
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

	opIDKey  = "opID"
	trimList = []string{"pkg"}
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
	return logger{
		dict:  []dict{},
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

func SetOpIDKey(key string) {
	opIDKey = key
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

func (l logger) Err(err error) OCMLogger {
	l.err = err
	return l
}

func (l logger) ExtraDeepReflect(name string, values any) OCMLogger {
	l.dict = append(l.dict, dict{
		name:   name,
		values: values,
	})
	return l
}

// AdditionalCallLevelSkips - allows to skip additional frames when logging, useful for wrapping loggers like OcmSdkLogWrapper
func (l logger) AdditionalCallLevelSkips(skip int) OCMLogger {
	l.additionalCallLevelSkips = skip
	return l
}

func (l logger) Extra(key string, value any) OCMLogger {
	l.extra[key] = value
	return l
}

func (l logger) ClearExtras() OCMLogger {
	l.extra = make(extra)
	return l
}

func (l logger) Info(args ...any) {
	l.log(zerolog.InfoLevel, sentry.LevelInfo, false, args...)
}

func (l logger) Debug(args ...any) {
	l.log(zerolog.DebugLevel, sentry.LevelDebug, false, args...)
}

func (l logger) Trace(args ...any) {
	l.log(zerolog.TraceLevel, sentry.LevelDebug, false, args...)
}

func (l logger) Warning(args ...any) {
	l.log(zerolog.WarnLevel, sentry.LevelWarning, false, args...)
}

func (l logger) Error(args ...any) {
	l.log(zerolog.ErrorLevel, sentry.LevelError, true, args...)
}

func (l logger) Fatal(args ...any) {
	l.log(zerolog.FatalLevel, sentry.LevelFatal, true, args...)
}

// Note: use the various "Depth" logging functions, so we get the correct file/line number in the logs
func (l logger) log(level zerolog.Level, sentryLevel sentry.Level, captureSentry bool, args ...any) {
	defer func() {
		l.err = nil
		l.ClearExtras()
	}()

	event := l.hydrateLog(level)
	message := ""
	if len(args) > 0 {
		message = args[0].(string)
		args = args[1:]
	}

	if captureSentry && (sentryLevel == sentry.LevelError || sentryLevel == sentry.LevelFatal) {
		if message == "" && l.err != nil {
			message = l.err.Error()
		}

		l.captureSentryEvent(sentryLevel, message, args...)
	}

	if event.Enabled() {
		event.Msgf(message, args...)
		event.Discard()
	}
}

func (l logger) captureSentryEvent(level sentry.Level, message string, args ...any) {
	// add extras to tags
	tags := make(tags)
	for k, v := range l.extra {
		tags[k] = fmt.Sprint(v) // tags can be strings only
	}
	event := sentry.NewEvent()
	event.Level = level
	event.Message = fmt.Sprintf(message, args...)
	event.Fingerprint = []string{getMD5Hash(event.Message)}
	event.Extra = l.extra
	event.Tags = tags
	if level == sentry.LevelError || level == sentry.LevelFatal {
		sentryStack := sentry.NewStacktrace()
		// Remove from the stack trace all the top frames that refer to this package, as those are a useless noise:
		stackSize := len(sentryStack.Frames)
		for stackSize > 0 {
			module := sentryStack.Frames[stackSize-1].Module
			if !strings.HasSuffix(module, "/pkg/logger") {
				break
			}
			stackSize--
		}
		sentryStack.Frames = sentryStack.Frames[:stackSize]
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
		log.Warn().Msg("Sentry hub does not present in logger")
		return
	}
	eventId := sentryHub.CaptureEvent(event)
	if eventId == nil {
		log.Error().Str("level", string(level)).Str("message", event.Message).Msg("Failed to capture sentry event")
	} else {
		log.Info().Msgf("Captured sentry event: %s", *eventId)
	}
}

func (l logger) hydrateLog(level zerolog.Level) *zerolog.Event {
	event := rootLogger.WithLevel(level).
		Caller(baseCallerSkipLevel + l.additionalCallLevelSkips).
		Err(l.err)

	if txid, ok := l.ctx.Value("txid").(int64); ok {
		event.Int64("tx_id", txid)
	}
	accountID := l.ctx.Value("accountID")
	if accountID != nil && accountID != "" {
		event.Str("accountID", fmt.Sprintf("%v", accountID))
	}
	if opid, ok := l.ctx.Value(opIDKey).(string); ok {
		event.Str("opid", opid)
	}

	if len(l.extra) > 0 {
		dict := zerolog.Dict()
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
			default:
				dict.Str(k, fmt.Sprint(v))
			}
		}
		event.Dict("Extra", dict)
	}

	// Would be nice to deprecate, reflection is expensive, but we need it to add structs like `http.Request` or `http.Response` etc.
	for _, dict := range l.dict {
		event.Interface(dict.name, dict.values)
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
