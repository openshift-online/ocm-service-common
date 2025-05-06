package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultCacheSize       = 1000
	defaultCacheExpireTime = time.Hour * 24
)

var requestsDispatched = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "region_proxy_dispatch_total",
		Help: "The total number of requests dispatched by region proxy.",
	},
)

type getClusterIdsHandlerFunc func(context.Context, logging.Logger, *http.Request) ClusterIds

type checkLocalHandlerFunc func(context.Context, logging.Logger, ClusterIds) (bool, error)

type dispatchHandlerFunc func(context.Context, logging.Logger, http.ResponseWriter, *http.Request,
	http.Handler, string) error

type errorHandlerFunc func(http.ResponseWriter, *http.Request, error)

type ClusterIds struct {
	Id         string
	ExternalId string
}

// Configuration for the region proxy middleware
//   - logger: The logger used in middleware.
//   - connection: The OCM SDK connection to use for the middleware.
//   - clusterCache: the cache stored cluster's region info
//   - getClusterIdsHandler: the function to retrieve cluster id/external_id from request
//   - checkLocalHandler: the function to check whether cluster is located in local server
//   - dispatchHandler: the function to dispatch the request
//   - errorHandler: The optional function to handle the error
type RegionProxy struct {
	logger               logging.Logger
	connection           *sdk.Connection
	clusterCache         *expirable.LRU[string, string]
	getClusterIdsHandler getClusterIdsHandlerFunc
	checkLocalHandler    checkLocalHandlerFunc
	dispatchHandler      dispatchHandlerFunc
	errorHandler         errorHandlerFunc
}

func init() {
	prometheus.MustRegister(requestsDispatched)
}

func NewRegionProxy(ctx context.Context, options ...RegionProxyMiddwareOption) *RegionProxy {

	regionProxyMiddleware := &RegionProxy{}
	for _, option := range options {
		option(regionProxyMiddleware)
	}

	if regionProxyMiddleware.clusterCache == nil {
		regionProxyMiddleware.clusterCache = expirable.NewLRU[string,
			string](defaultCacheSize, nil, defaultCacheExpireTime)
	}

	if regionProxyMiddleware.logger == nil {
		regionProxyMiddleware.logger, _ = sdk.NewGoLoggerBuilder().
			Info(true).
			Build()
	}

	if regionProxyMiddleware.dispatchHandler == nil {
		regionProxyMiddleware.dispatchHandler = defaultDispatchHandler()
	}

	if regionProxyMiddleware.errorHandler == nil {
		regionProxyMiddleware.errorHandler = defaultErrorHandler()
	}

	return regionProxyMiddleware
}

func (rp *RegionProxy) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx := r.Context()
		ids := rp.getClusterIdsHandler(ctx, rp.logger, r)
		if ids.Id == "" && ids.ExternalId == "" {
			rp.logger.Warn(ctx, "Cannot find cluster id or external id from request")
			next.ServeHTTP(w, r)
			return
		}

		rhRegionId, found := rp.checkCache(ids.Id, ids.ExternalId)
		if !found {
			if rp.checkLocalHandler != nil {
				exists, err := rp.checkLocalHandler(ctx, rp.logger, ids)
				if err != nil {
					rp.errorHandler(w, r, err)
					return
				}
				if exists {
					rp.updateCache("", ids.Id, ids.ExternalId)
					next.ServeHTTP(w, r)
					return
				}
			}
			rhRegionId, err = getRhRegionId(ctx, rp.logger, ids, rp.connection)
			rp.updateCache(rhRegionId, ids.Id, ids.ExternalId)
			if err != nil {
				rp.errorHandler(w, r, err)
				return
			}
		}

		err = rp.dispatchHandler(ctx, rp.logger, w, r, next, rhRegionId)
		if err != nil {
			rp.errorHandler(w, r, err)
		}
	})
}

func defaultDispatchHandler() dispatchHandlerFunc {
	return func(ctx context.Context, logger logging.Logger, w http.ResponseWriter, r *http.Request,
		next http.Handler, rhRegionId string) error {
		if rhRegionId != "" {
			// "rh_region_id":"aws.ap-southeast-1.integration" => api.aws.ap-southeast-1.integration.openshift.com
			// "rh_region_id":"aws.ap-southeast-1.stage" => api.aws.ap-southeast-1.stage.openshift.com
			// "rh_region_id":"aws.ap-southeast-1" => api.aws.ap-southeast-1.openshift.com
			dispatchHost := fmt.Sprintf("https://api.%s.openshift.com", rhRegionId)
			logger.Info(ctx, "Dispatch the request %s to %s", r.URL, dispatchHost)
			requestsDispatched.Inc()
			dispatchURL, err := url.Parse(dispatchHost)
			if err != nil {
				return err
			}
			if dispatchURL.Scheme == "" {
				dispatchURL = &url.URL{
					Host:   dispatchHost,
					Scheme: "https"}
			}
			r.Host = dispatchURL.Host
			proxy := httputil.NewSingleHostReverseProxy(dispatchURL)
			proxy.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
		return nil
	}
}

func defaultErrorHandler() errorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error in region proxy: %v", err)))
	}
}

func getRhRegionId(ctx context.Context, logger logging.Logger,
	ids ClusterIds, connection *sdk.Connection) (string, error) {
	var search string
	if ids.Id != "" {
		search = fmt.Sprintf("cluster_id='%s'", ids.Id)
	} else {
		search = fmt.Sprintf("external_cluster_id='%s'", ids.ExternalId)
	}
	resp, err := connection.AccountsMgmt().V1().Subscriptions().List().Search(search).SendContext(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to list cluster in AMS: %v", err)
		return "", err
	}
	if resp.Items().Len() > 0 {
		return resp.Items().Get(0).RhRegionID(), nil
	}
	return "", nil
}

func (rp *RegionProxy) checkCache(ids ...string) (string, bool) {
	for _, id := range ids {
		value, found := rp.clusterCache.Get(id)
		if found {
			return value, found
		}
	}
	return "", false
}

func (rp *RegionProxy) updateCache(rhRegionID string, ids ...string) {
	for _, id := range ids {
		if id != "" {
			rp.clusterCache.Add(id, rhRegionID)
		}
	}
}
