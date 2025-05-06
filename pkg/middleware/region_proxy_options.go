package middleware

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type RegionProxyMiddwareOption func(*RegionProxy)

func WithProxyLogger(logger logging.Logger) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.logger = logger
	}
}

func WithSDKConnection(conn *sdk.Connection) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.connection = conn
	}
}

func WithGetClusterIdsHandler(fn getClusterIdsHandlerFunc) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.getClusterIdsHandler = fn
	}
}

func WithCheckLocalHandler(fn checkLocalHandlerFunc) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.checkLocalHandler = fn
	}
}

func WithDispatchHandler(fn dispatchHandlerFunc) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.dispatchHandler = fn
	}
}

func WithErrorHandler(fn errorHandlerFunc) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.errorHandler = fn
	}
}

func WithClusterCache(size int, expireTime time.Duration) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.clusterCache = expirable.NewLRU[string, string](size, nil, expireTime)
	}
}
