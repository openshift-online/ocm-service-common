package segment

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/segmentio/analytics-go/v3"

	logger "gitlab.cee.redhat.com/service/ocm-common/pkg/ocmlogger"
)

type Service interface {
	Track(ctx context.Context, event string, ocmResourceType string, initiator string, subscription *ApiSubscription, creatorRhitWebUserId *string) error
	TrackAccount(ctx context.Context, account *Account, isOrgAdmin *bool) error
}

type TrackService service

var _ Service = &TrackService{}

func TrackSegment(ctx context.Context, event string, ocmResourceType string, initiator string) {
	ulog := logger.NewOCMLogger(ctx)
	client := getSegmentClientFromContext(ctx)
	go func() {
		if client != nil {
			err := client.TrackService.Track(ctx, event, ocmResourceType, initiator, nil, nil)
			if err != nil {
				ulog.Contextual().Error(err, "Segment Track",
					"Event Name", event, "ocmResourceType", ocmResourceType, "initiator", initiator,
				)
			}
		}
	}()
}

func TrackSegmentWithSubscription(ctx context.Context, event string, ocmResourceType string, initiator string, subscription *ApiSubscription, creatorRhitWebUserId *string) {
	client := getSegmentClientFromContext(ctx)
	ulog := logger.NewOCMLogger(ctx)
	go func() {
		if client != nil {
			ulog.Info("Track event=%s ocmResourceType=%s initiator=%s creatorRhitWebUserId=%v subscription=%v", event, ocmResourceType, initiator, creatorRhitWebUserId, subscription)
			err := client.TrackService.Track(ctx, event, ocmResourceType, initiator, subscription, creatorRhitWebUserId)
			if err != nil {
				ulog.Contextual().Error(err, "Segment Track",
					"Event Name", event, "ocmResourceType", ocmResourceType, "initiator", initiator,
				)
			}
		}
	}()
}

func TrackSegmentWithAccount(ctx context.Context, account *Account, isOrgAdmin *bool) {
	client := getSegmentClientFromContext(ctx)
	ulog := logger.NewOCMLogger(ctx)
	go func() {
		if client != nil {
			ulog.Info("TrackAccount account=%v", account)
			err := client.TrackService.TrackAccount(ctx, account, isOrgAdmin)
			if err != nil {
				ulog.Contextual().Error(err, "Segment TrackAccount", "account", account)
			}
		}
	}()
}

func (s *TrackService) TrackAccount(ctx context.Context, account *Account, isOrgAdmin *bool) error {
	if s.config.Key == "" || account == nil {
		return nil
	}

	connection, err := analytics.NewWithConfig(s.config.Key, s.aConfig)
	if err != nil {
		return err
	}
	defer connection.Close()

	webUserID := s.getWebUserId(ctx, account.RhitWebUserId)
	if webUserID == "" {
		// Do not track events of an unrecognizable account
		return nil
	}
	contxt := setContext(ctx, s.config.Version)

	traits := analytics.Traits{}
	if account.Email != "" {
		traits.Set("email", account.Email)
		traits.Set("internal", redhatIBMEmailRe.MatchString(account.Email))
		if _, emailDomain, found := strings.Cut(account.Email, "@"); found {
			traits.Set("email_domain", emailDomain)
		}
	}
	if isOrgAdmin != nil {
		traits.Set("isOrgAdmin", *isOrgAdmin)
	}
	name := account.FirstName
	if account.LastName != "" {
		name += " " + account.LastName
	}
	traits.Set("name", name)

	if err = connection.Enqueue(analytics.Identify{
		UserId:  webUserID,
		Traits:  traits,
		Context: contxt,
	}); err != nil {
		return err
	}

	traits = analytics.Traits{}
	if account.OrgEbsAccountID != nil {
		traits["cloud_ebs_id"] = account.OrgEbsAccountID
	}
	if err = connection.Enqueue(analytics.Group{
		UserId:  webUserID,
		GroupId: account.OrgExternalID,
		Context: contxt,
		Traits:  traits,
	}); err != nil {
		return err
	}

	return nil
}

func (s *TrackService) Track(ctx context.Context, event string, ocmResourceType string, initiator string, subscription *ApiSubscription, creatorRhitWebUserId *string) error {
	if s.config.Key == "" {
		return nil
	}

	trackParams := s.getTrackParams(ctx, event, ocmResourceType, initiator, creatorRhitWebUserId)
	if trackParams.WebUserID == "" {
		// Do not track events of an unrecognizable account
		return nil
	}

	setResourceAndSubscription(trackParams, subscription)

	properties := analytics.NewProperties().
		Set("initiator", strings.ToLower(trackParams.Initiator)).
		Set("ocm_resource_type", strings.ToLower(trackParams.OcmResourceType)).
		Set("error", false)

	if trackParams.Subscription.SupportLevel == "Eval" {
		trackParams.Subscription.SupportLevel = ""
	}
	if trackParams.Subscription.ServiceLevel == "Eval" {
		trackParams.Subscription.ServiceLevel = ""
	}
	if trackParams.Subscription.SystemUnits == "Eval" {
		trackParams.Subscription.SystemUnits = ""
	}
	if trackParams.Subscription.Usage == "Eval" {
		trackParams.Subscription.Usage = ""
	}

	fields := []struct {
		name  string
		value interface{}
	}{
		{"resource_id", trackParams.Resource.Id},
		{"resource_created_at", trackParams.Resource.CreatedAt},
		{"resource_provenance", trackParams.Resource.Provenance},
		{"resource_status", trackParams.Resource.Status},
		{"resource_managed", trackParams.Resource.Managed},
		{"subscription_billing_model", trackParams.Subscription.BillingModel},
		{"subscription_support_level", trackParams.Subscription.SupportLevel},
		{"subscription_service_level", trackParams.Subscription.ServiceLevel},
		{"subscription_usage", trackParams.Subscription.Usage},
		{"subscription_system_units", trackParams.Subscription.SystemUnits},
		{"subscription_core_count", trackParams.Subscription.CpuTotal},
		{"subscription_socket_count", trackParams.Subscription.SocketTotal},
	}
	for _, fld := range fields {
		switch v := fld.value.(type) {
		case int:
		case int32:
		case int64:
			if v != 0 {
				properties = properties.Set(fld.name, v)
			}
		case string:
			if v != "" {
				properties = properties.Set(fld.name, strings.ToLower(v))
			}
		case bool:
			properties = properties.Set(fld.name, v)
		default:
			if v, ok := fld.value.(time.Time); ok {
				if !v.IsZero() {
					properties = properties.Set(fld.name, v)
				}
			}
		}
	}

	connection, err := analytics.NewWithConfig(s.config.Key, s.aConfig)
	if err != nil {
		return err
	}
	defer connection.Close()

	err = connection.Enqueue(analytics.Track{
		UserId:     trackParams.WebUserID,
		Event:      event,
		Properties: properties,
		Context:    setContext(ctx, s.config.Version),
	})
	// Update metrics
	s.client.track(err)
	return err
}

func setContext(ctx context.Context, version string) *analytics.Context {
	var (
		ua, locale, groupID string
		ip                  net.IP
		ok                  bool
	)
	if groupID, ok = ctx.Value(contextRHITOrgId).(string); !ok {
		groupID = ""
	}
	if ip, ok = ctx.Value(contextRequestIPkey).(net.IP); !ok {
		ip = net.IP{}
	}
	if ua, ok = ctx.Value(contextRequestUAkey).(string); !ok {
		ua = ""
	}
	if locale, ok = ctx.Value(contextRHITUserLocale).(string); !ok {
		locale = ""
	}
	return &analytics.Context{
		App: analytics.AppInfo{
			Name:    "OCM Account Manager",
			Version: version,
		},
		IP:        ip,
		UserAgent: ua,
		Locale:    locale,
		Extra: map[string]interface{}{
			"groupId": groupID,
		},
	}
}

// Empty returns an empty value of type T.
func Empty[T any]() T {
	var zero T
	return zero
}

// FromPtr returns the pointer value or empty.
func FromPtr[T any](v *T) T {
	if v == nil {
		return Empty[T]()
	}
	return *v
}

func NilToZeroInt32(a *int) *int32 {
	var res int32 = 0
	if a != nil {
		res = int32(*a)
	}
	return &res
}

func setResourceAndSubscription(trackParams *TrackParams, subscription *ApiSubscription) {
	if subscription == nil {
		return
	}
	trackParams.Resource = Resource{
		Id:         subscription.ExternalClusterID,
		CreatedAt:  subscription.CreatedAt,
		Provenance: FromPtr(subscription.Provenance),
		Status:     subscription.Status,
		Managed:    subscription.Managed,
	}
	trackParams.Subscription = Subscription{
		BillingModel: FromPtr(subscription.ClusterBillingModel),
		SupportLevel: FromPtr(subscription.SupportLevel),
		ServiceLevel: FromPtr(subscription.ServiceLevel),
		Usage:        FromPtr(subscription.Usage),
		SystemUnits:  FromPtr(subscription.SystemUnits),
		CpuTotal:     *NilToZeroInt32(subscription.CpuTotal),
		SocketTotal:  *NilToZeroInt32(subscription.SocketTotal),
	}
}

func (s *TrackService) getWebUserId(ctx context.Context, creatorRhitWebUserId *string) string {
	return s.client.getUserId(ctx, creatorRhitWebUserId)
}

func (s *TrackService) getTrackParams(ctx context.Context, event string, ocmResourceType string, initiator string, creatorRhitWebUserId *string) *TrackParams {
	webUserID := s.getWebUserId(ctx, creatorRhitWebUserId)

	return &TrackParams{
		WebUserID:       webUserID,
		OcmResourceType: ocmResourceType,
		Initiator:       initiator,
		Event:           event,
	}
}
