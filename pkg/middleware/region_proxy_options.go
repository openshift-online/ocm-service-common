package middleware

import (
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type RegionProxyMiddwareOption func(*RegionProxy)

func WithProxyLogger(logger logging.Logger) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.Logger = logger
	}
}

func WithSDKConnection(conn *sdk.Connection) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.Connection = conn
	}
}

func WithGetDispatchHostFunc(fn getDispatchHostFunc) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.GetDispatchHostFunc = fn
	}
}

func WithErrorHandler(fn errorHandler) RegionProxyMiddwareOption {
	return func(middleware *RegionProxy) {
		middleware.ErrorHandler = fn
	}
}
