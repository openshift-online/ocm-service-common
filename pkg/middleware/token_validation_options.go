package middleware

import (
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type TokenScopeValidationMiddwareOption func(*TokenScopeValidationMiddlewareImpl)

func WithConnection(conn *sdk.Connection) TokenScopeValidationMiddwareOption {
	return func(middleware *TokenScopeValidationMiddlewareImpl) {
		middleware.Connection = conn
	}
}

func WithPollingInterval(pollingInterval time.Duration) TokenScopeValidationMiddwareOption {
	return func(middleware *TokenScopeValidationMiddlewareImpl) {
		middleware.PollingIntervalOverride = pollingInterval
	}
}

func WithCallback(fn callback) TokenScopeValidationMiddwareOption {
	return func(middleware *TokenScopeValidationMiddlewareImpl) {
		middleware.CallbackFn = fn
	}
}

func WithLogger(logger logging.Logger) TokenScopeValidationMiddwareOption {
	return func(middleware *TokenScopeValidationMiddlewareImpl) {
		middleware.Logger = logger
	}
}

func WithErrorOnMissingToken() TokenScopeValidationMiddwareOption {
	return func(middleware *TokenScopeValidationMiddlewareImpl) {
		middleware.ErrorOnMissingToken = true
	}
}
