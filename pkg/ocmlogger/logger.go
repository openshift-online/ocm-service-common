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
	// Extra stores a key-value pair in a map; entry with the same key will be overwritten
	// All simple (non-struct, non-slice, non-map etc.) values will be also send to sentry/glitchtip as `tags`
	Extra(key string, value any) OCMLogger
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

type logger struct {
	ctx                      context.Context
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

	trimList = []string{"pkg"}

	opIDCallback      func(ctx context.Context) string
	accountIDCallback func(ctx context.Context) string
	txIDCallback      func(ctx context.Context) int64
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

func SetOpIDCallback(getOpId func(ctx context.Context) string) {
	opIDCallback = getOpId
}

func SetAccountIDCallback(getAccountId func(ctx context.Context) string) {
	accountIDCallback = getAccountId
}

func SetTxIDCallback(getTxId func(ctx context.Context) int64) {
	txIDCallback = getTxId
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
	l.extra[key] = value
	return l
}

func (l *logger) ClearExtras() OCMLogger {
	l.extra = make(extra)
	return l
}

func (l *logger) Info(args ...any) {
	l.log(zerolog.InfoLevel, sentry.LevelInfo, false, args...)
}

func (l *logger) Debug(args ...any) {
	l.log(zerolog.DebugLevel, sentry.LevelDebug, false, args...)
}

func (l *logger) Trace(args ...any) {
	l.log(zerolog.TraceLevel, sentry.LevelDebug, false, args...)
}

func (l *logger) Warning(args ...any) {
	l.log(zerolog.WarnLevel, sentry.LevelWarning, false, args...)
}

func (l *logger) Error(args ...any) {
	l.log(zerolog.ErrorLevel, sentry.LevelError, true, args...)
}

func (l *logger) Fatal(args ...any) {
	l.log(zerolog.FatalLevel, sentry.LevelFatal, true, args...)
}

// Note: use the various "Depth" logging functions, so we get the correct file/line number in the logs
func (l *logger) log(level zerolog.Level, sentryLevel sentry.Level, captureSentry bool, args ...any) {
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

func (l *logger) captureSentryEvent(level sentry.Level, message string, args ...any) {
	event := sentry.NewEvent()
	event.Level = level
	event.Message = fmt.Sprintf(message, args...)
	event.Fingerprint = []string{getMD5Hash(event.Message)}
	event.Extra = l.extra
	// add simple extras to tags
	event.Tags = make(tags)
	for k, v := range l.extra {
		switch v.(type) {
		case string:
			event.Tags[k] = v.(string)
		case bool:
			event.Tags[k] = "false"
			if v.(bool) {
				event.Tags[k] = "true"
			}
		case int:
			i := v.(int)
			event.Tags[k] = strconv.FormatInt(int64(i), 10)
		case int8:
			i := v.(int8)
			event.Tags[k] = strconv.FormatInt(int64(i), 10)
		case int16:
			i := v.(int16)
			event.Tags[k] = strconv.FormatInt(int64(i), 10)
		case int32:
			i := v.(int32)
			event.Tags[k] = strconv.FormatInt(int64(i), 10)
		case int64:
			event.Tags[k] = strconv.FormatInt(v.(int64), 10)
		case float32:
			event.Tags[k] = strconv.FormatFloat(float64(v.(float32)), 'f', 2, 32)
		case float64:
			event.Tags[k] = strconv.FormatFloat(v.(float64), 'f', 2, 64)
		default:
			// skip complex types
		}
	}
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

func (l *logger) hydrateLog(level zerolog.Level) *zerolog.Event {
	event := rootLogger.WithLevel(level).
		Caller(baseCallerSkipLevel + l.additionalCallLevelSkips).
		Err(l.err)

	if txIDCallback != nil {
		txid := txIDCallback(l.ctx)
		if txid != 0 {
			event.Int64("tx_id", txid)
		}
	}
	if accountIDCallback != nil {
		accountID := accountIDCallback(l.ctx)
		if accountID != "" {
			event.Str("accountID", accountID)
		}
	}
	if opIDCallback != nil {
		opid := opIDCallback(l.ctx)
		if opid != "" {
			event.Str("opid", opid)
		}
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
			case float32:
				dict.Float32(k, typ)
			case float64:
				dict.Float64(k, typ)
			default:
				// Note: reflection is expensive, but we need it to add structs like `http.Request` or `http.Response` etc.
				dict.Any(k, v)
			}
		}
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
