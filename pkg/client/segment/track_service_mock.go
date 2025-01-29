package segment

import (
	"context"

	logger "github.com/openshift-online/ocm-service-common/pkg/ocmlogger"
)

type TrackServiceMock service

var _ Service = &TrackServiceMock{}

func (s *TrackServiceMock) Track(ctx context.Context, event string, ocmResource, initiator string, subscription *ApiSubscription, creatorRhitWebUserId *string) error {
	webUserID := s.client.getUserId(ctx, creatorRhitWebUserId)
	ulog := logger.NewOCMLogger(ctx)
	ulog.Warning("Tracked account %s; event name %s", webUserID, event)
	return nil
}

func (s *TrackServiceMock) TrackAccount(ctx context.Context, account *Account, _ *bool) error {
	webUserID := s.client.getUserId(ctx, account.RhitWebUserId)
	ulog := logger.NewOCMLogger(ctx)
	ulog.Warning("Tracked account %s Account %s", webUserID, account.Email)
	return nil
}
