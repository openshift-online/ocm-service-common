package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/prometheus/client_golang/prometheus"
)

var requestsDispatched = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "region_proxy_dispatch_total",
		Help: "The total number of requests dispatched by region proxy.",
	},
)

type getDispatchHostFunc func(context.Context, logging.Logger, *http.Request, *sdk.Connection) (string, error)

type errorHandler func(http.ResponseWriter, *http.Request, error)

// Configuration for the region proxy middleware
//   - Logger: The logger used in middleware.
//   - Connection: The OCM SDK connection to use for the middleware.
//   - GetDispatchHostFunc: The function to get the host where request to be dispatched.
//   - ErrorHandler: The optional function to handle the error.
type RegionProxy struct {
	Logger              logging.Logger
	Connection          *sdk.Connection
	GetDispatchHostFunc getDispatchHostFunc
	ErrorHandler        errorHandler
}

func init() {
	prometheus.MustRegister(requestsDispatched)
}

func NewRegionProxy(ctx context.Context, options ...RegionProxyMiddwareOption) *RegionProxy {

	regionProxyMiddleware := &RegionProxy{}
	for _, option := range options {
		option(regionProxyMiddleware)
	}

	if regionProxyMiddleware.Logger == nil {
		regionProxyMiddleware.Logger, _ = sdk.NewGoLoggerBuilder().
			Info(true).
			Build()
	}

	if regionProxyMiddleware.GetDispatchHostFunc == nil {
		regionProxyMiddleware.Logger.Warn(ctx, "GetDispatchHostFunc is not defined for the region proxy")
	}

	if regionProxyMiddleware.ErrorHandler == nil {
		regionProxyMiddleware.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error in region proxy: %v", err)))
		}
	}

	return regionProxyMiddleware
}

func (rp *RegionProxy) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rp.GetDispatchHostFunc == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		dispatchHost := ""
		dispatchHost, err := rp.GetDispatchHostFunc(r.Context(), rp.Logger, r, rp.Connection)
		if err != nil {
			rp.ErrorHandler(w, r, err)
			return
		}

		if dispatchHost != "" {
			rp.Logger.Info(ctx, "Dispatch the request %s to %s", r.URL, dispatchHost)
			requestsDispatched.Inc()
			r.Host = dispatchHost
			dispatchURL, _ := url.Parse(fmt.Sprintf("https://%s", dispatchHost))
			proxy := httputil.NewSingleHostReverseProxy(dispatchURL)
			proxy.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
