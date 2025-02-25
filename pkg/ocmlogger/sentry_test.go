package ocmlogger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("OCMLogger Sentry Integration", Label("logger"), func() {
	var (
		ulog            OCMLogger
		output          bytes.Buffer
		sentryTransport *TransportMock
		sentryClient    *sentry.Client
		ctx             context.Context
	)

	BeforeEach(func() {
		output = bytes.Buffer{}
		SetOutput(WrapUnsafeWriterWithLocks(&output))
		DeferCleanup(func() {
			sentryTransport.Flush(0)
			SetOutput(os.Stderr)
		})
		sentryTransport = &TransportMock{}
		sentryClient, _ = sentry.NewClient(sentry.ClientOptions{
			Dsn:       "http://whatever@example.com/1337",
			Transport: sentryTransport,
			Integrations: func(i []sentry.Integration) []sentry.Integration {
				return []sentry.Integration{}
			},
		})
		ctx = sentry.SetHubOnContext(context.Background(), sentry.NewHub(sentryClient, sentry.NewScope()))
		ulog = NewOCMLogger(ctx)
	})

	Context("Error", func() {
		It("Below Error level does not publish sentry events by default", func() {
			ulog.Warning("warning")
			Expect(sentryTransport.lastEvent).To(BeNil())
			ulog.Info("info")
			Expect(sentryTransport.lastEvent).To(BeNil())
			ulog.Debug("debug")
			Expect(sentryTransport.lastEvent).To(BeNil())
			ulog.Trace("trace")
			Expect(sentryTransport.lastEvent).To(BeNil())
		})

		It("Error level publishes sentry events by default", func() {
			ulog.Error("ERROR")
			Expect(sentryTransport.lastEvent).NotTo(BeNil())
			Expect(sentryTransport.lastEvent.Message).To(Equal("ERROR"))
		})

		It("Error level does not publish events when overridden", func() {
			ulog.CaptureSentryEvent(false).Error("ERROR")
			Expect(sentryTransport.lastEvent).To(BeNil())
		})

		It("Below Error level publishes sentry events when overridden", func() {
			ulog.CaptureSentryEvent(true).Warning("warning")
			Expect(sentryTransport.lastEvent).NotTo(BeNil())
			Expect(sentryTransport.lastEvent.Message).To(Equal("warning"))
		})

		It("Adds SentryEventID to log", func() {
			ulog.Error("ERROR")
			result := output.String()
			Expect(result).To(ContainSubstring("\"SentryEventID\""))
		})

		It("Uses error stacktrace when possible", func() {
			err := errors.New("github.com/pkg/errors have stacktraces")
			expectedLineNumber := 81 // ^^^

			ulog.Err(err).Error("ERROR")

			Expect(sentryTransport.lastEvent).NotTo(BeNil())
			Expect(sentryTransport.lastEvent.Exception).To(HaveLen(1))
			Expect(sentryTransport.lastEvent.Exception[0].Stacktrace).NotTo(BeNil())

			stacktrace := sentryTransport.lastEvent.Exception[0].Stacktrace
			lastFrame := stacktrace.Frames[len(stacktrace.Frames)-1]

			//Any file modification risks breaking this test, I dont love it, but I cant think of a better way
			Expect(lastFrame.AbsPath).To(ContainSubstring("ocm-common/pkg/ocmlogger/sentry_test.go"))
			Expect(lastFrame.Lineno).To(Equal(expectedLineNumber))
		})

		It("generates a new stacktrace", func() {
			err := fmt.Errorf("This kind of error does not generate a stacktrace")

			ulog.Err(err).Error("ERROR")
			expectedLineNumber := 100 // ^^^
			Expect(sentryTransport.lastEvent).NotTo(BeNil())
			Expect(sentryTransport.lastEvent.Exception).To(HaveLen(1))
			Expect(sentryTransport.lastEvent.Exception[0].Stacktrace).NotTo(BeNil())

			stacktrace := sentryTransport.lastEvent.Exception[0].Stacktrace
			lastFrame := stacktrace.Frames[len(stacktrace.Frames)-1]

			//Any file modification risks breaking this test, I dont love it, but I cant think of a better way
			Expect(lastFrame.AbsPath).To(ContainSubstring("ocm-common/pkg/ocmlogger/sentry_test.go"))
			Expect(lastFrame.Lineno).To(Equal(expectedLineNumber))
		})
	})
})

/**
 * Mocks, inspired by sentry-go's own tests
 */
type TransportMock struct {
	mu        sync.Mutex
	events    []*sentry.Event
	lastEvent *sentry.Event
}

func (t *TransportMock) Configure(_ sentry.ClientOptions) {}
func (t *TransportMock) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
	t.lastEvent = event
}
func (t *TransportMock) Flush(_ time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = make([]*sentry.Event, 0)
	t.lastEvent = nil
	return true
}
func (t *TransportMock) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events
}
