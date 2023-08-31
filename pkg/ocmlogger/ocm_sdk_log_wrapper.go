package ocmlogger

import (
	"context"

	sdk "github.com/openshift-online/ocm-sdk-go"
)

/**
 * This is a wrapper around OCMLogger that implements the ocm-sdk logging interface.
 */

type OcmSdkLogWrapper struct{}

var _ sdk.Logger = &OcmSdkLogWrapper{}

func NewOcmSdkLogWrapper() *OcmSdkLogWrapper {
	return &OcmSdkLogWrapper{}
}

func (w *OcmSdkLogWrapper) DebugEnabled() bool {
	return DebugEnabled()
}

func (w *OcmSdkLogWrapper) InfoEnabled() bool {
	return InfoEnabled()
}

func (w *OcmSdkLogWrapper) WarnEnabled() bool {
	return WarnEnabled()
}

func (w *OcmSdkLogWrapper) ErrorEnabled() bool {
	return ErrorEnabled()
}

func (w *OcmSdkLogWrapper) Debug(ctx context.Context, format string, args ...interface{}) {
	params := []interface{}{format}
	params = append(params, args...)
	NewOCMLogger(ctx).AdditionalCallLevelSkips(1).Debug(params...)
}

func (w *OcmSdkLogWrapper) Info(ctx context.Context, format string, args ...interface{}) {
	params := []interface{}{format}
	params = append(params, args...)
	NewOCMLogger(ctx).AdditionalCallLevelSkips(1).Info(params...)
}

func (w *OcmSdkLogWrapper) Warn(ctx context.Context, format string, args ...interface{}) {
	params := []interface{}{format}
	params = append(params, args...)
	NewOCMLogger(ctx).AdditionalCallLevelSkips(1).Warning(params...)
}

func (w *OcmSdkLogWrapper) Error(ctx context.Context, format string, args ...interface{}) {
	params := []interface{}{format}
	params = append(params, args...)
	NewOCMLogger(ctx).AdditionalCallLevelSkips(1).Error(params...)
}

func (w *OcmSdkLogWrapper) Fatal(ctx context.Context, format string, args ...interface{}) {
	params := []interface{}{format}
	params = append(params, args...)
	NewOCMLogger(ctx).Fatal(params...)
}
