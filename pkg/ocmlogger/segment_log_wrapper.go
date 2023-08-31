package ocmlogger

import (
	"context"

	segment "github.com/segmentio/analytics-go/v3"
)

/**
 * SegmentLogWrapper is a wrapper around OCMLogger that implements the segment.Logger interface.
 */

type SegmentLogWrapper struct{}

var _ segment.Logger = &SegmentLogWrapper{}

func NewSegmentLogWrapper() *SegmentLogWrapper {
	return &SegmentLogWrapper{}
}

func (w *SegmentLogWrapper) Logf(format string, args ...interface{}) {
	NewOCMLogger(context.Background()).AdditionalCallLevelSkips(1).Info(format, args)
}

func (w *SegmentLogWrapper) Errorf(format string, args ...interface{}) {
	NewOCMLogger(context.Background()).AdditionalCallLevelSkips(1).Error(format, args)
}
