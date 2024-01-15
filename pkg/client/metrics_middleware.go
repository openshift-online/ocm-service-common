package client

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus"
)

// metrics name and labels
const (
	MetricsSubsystem       = "api_outbound"
	MetricsAPIServiceLabel = "apiservice"
	MetricsCodeLabel       = "code"
	MetricsMethodLabel     = "method"
	MetricsPathLabel       = "path"
	PathVarSub             = "-"
)

type ServiceClient interface {
	// GetServiceName returns the name of the service
	GetServiceName() string
	// GetRouter returns the router with the routes
	GetRouter() *mux.Router
}

// AddMetricsMiddleware adds metrics middleware to the http client
func AddMetricsMiddleware(client ServiceClient, httpClient *http.Client) *http.Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	wrapped := http.DefaultTransport
	if httpClient.Transport != nil {
		wrapped = httpClient.Transport
	}
	httpClient.Transport = &metricsRoundTripperWrapper{
		serviceName:   client.GetServiceName(),
		serviceRouter: client.GetRouter(),
		wrapped:       wrapped,
	}
	return httpClient
}

// AddMetricsMiddlewareByTransport adds metrics middleware to the client's transport
// note: for uhc clients, we use ocm-sdk and don't have access to their http client.
func AddMetricsMiddlewareByTransport(client ServiceClient, transport http.RoundTripper) http.RoundTripper {
	wrapper := &metricsRoundTripperWrapper{
		serviceName:   client.GetServiceName(),
		serviceRouter: client.GetRouter(),
		wrapped:       transport,
	}
	return wrapper
}

type metricsRoundTripperWrapper struct {
	serviceName   string
	serviceRouter *mux.Router
	wrapped       http.RoundTripper
}

// RoundTrip calls the wrapped RoundTrip and update metrics
func (m *metricsRoundTripperWrapper) RoundTrip(request *http.Request) (*http.Response, error) {
	before := time.Now()
	response, err := m.wrapped.RoundTrip(request)
	after := time.Now()
	elapsed := after.Sub(before)
	path, serviceName := reducePath(m.serviceRouter, request)
	if serviceName == "" {
		// if no overrides, use the service name defined on client level
		serviceName = m.serviceName
	}
	code := 0
	if response != nil {
		code = response.StatusCode
	}
	updateMetrics(serviceName, request.Method, path, strconv.Itoa(code), elapsed.Seconds())

	return response, err
}

// regex to convert template param {id} to -
var metricsPathVarRE = regexp.MustCompile(`{[^}]*}`)

// labels added to metrics:
var metricsLabels = []string{
	MetricsAPIServiceLabel,
	MetricsCodeLabel,
	MetricsMethodLabel,
	MetricsPathLabel,
}

// count metric
var requestCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: MetricsSubsystem,
		Name:      "request_count",
		Help:      "Number of requests sent.",
	},
	metricsLabels,
)

// duration metric
var requestDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: MetricsSubsystem,
		Name:      "request_duration",
		Help:      "Request duration in seconds.",
		Buckets: []float64{
			0.1,
			1.0,
			2.0,
			5.0,
			10.0,
			30.0,
		},
	},
	metricsLabels,
)

// RegisterClientMetrics registers the metrics with the Prometheus library.
func RegisterClientMetrics(ctx context.Context) error {
	// Register the count metric:
	err := prometheus.Register(requestCountMetric)
	if err != nil {
		registered, ok := err.(prometheus.AlreadyRegisteredError)
		if ok {
			requestCountMetric = registered.ExistingCollector.(*prometheus.CounterVec)
		} else {
			return err
		}
	}

	// Register the duration metric:
	err = prometheus.Register(requestDurationMetric)
	if err != nil {
		registered, ok := err.(prometheus.AlreadyRegisteredError)
		if ok {
			requestDurationMetric = registered.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			return err
		}
	}

	return nil
}

func ResetClientMetrics() {
	requestCountMetric.Reset()
	requestDurationMetric.Reset()
}

func reducePath(router *mux.Router, request *http.Request) (path, serviceName string) {
	path = "/" + PathVarSub
	serviceName = ""
	if router != nil {
		matched := mux.RouteMatch{}
		ok := router.Match(request, &matched)
		if ok && matched.Route != nil {
			template, err := matched.Route.GetPathTemplate()
			if err == nil {
				path = metricsPathVarRE.ReplaceAllString(template, PathVarSub)
				serviceName = matched.Route.GetName()
			}
		} else {
			// use the 1st part of the route
			parts := strings.Split(request.URL.Path, "/")
			for _, part := range parts {
				if part != "" {
					path = "/" + part
					break
				}
			}
		}
	}
	return path, serviceName
}

func updateMetrics(apiService string, method string, path string, code string, durationInSecs float64) {

	labels := map[string]string{
		MetricsAPIServiceLabel: apiService,
		MetricsMethodLabel:     method,
		MetricsPathLabel:       path,
		MetricsCodeLabel:       code,
	}
	// update count metric
	requestCountMetric.With(labels).Inc()
	// update duration metric
	requestDurationMetric.With(labels).Observe(durationInSecs)
}
