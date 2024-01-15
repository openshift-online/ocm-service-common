package segment

import "time"

type AuthPayload struct {
	ClientID          string `json:"clientId"`
	RHITUserID        string `json:"user_id"`
	Locale            string `json:"locale"`
	OrgId             string `json:"org_id"`
	RHITAccountNumber string `json:"account_number"`
}

type Resource struct {
	Id         string
	CreatedAt  time.Time
	Provenance string
	Status     string
	Managed    bool
}

type ApiSubscription struct {
	CreatedAt           time.Time
	Provenance          *string
	Status              string
	Managed             bool
	ExternalClusterID   string
	ClusterBillingModel *string
	SupportLevel        *string
	ServiceLevel        *string
	Usage               *string
	SystemUnits         *string
	CpuTotal            *int
	SocketTotal         *int
}

type Subscription struct {
	BillingModel string
	SupportLevel string
	ServiceLevel string
	Usage        string
	SystemUnits  string
	CpuTotal     int32
	SocketTotal  int32
}

type TrackParams struct {
	WebUserID       string `json:"webUserId"`
	Event           string `json:"event"`
	OcmResourceType string `json:"ocm_resource_type"`
	Initiator       string `json:"initiator"`
	Resource        Resource
	Subscription    Subscription
}

type Account struct {
	RhitWebUserId   *string
	Email           string
	FirstName       string
	LastName        string
	OrgEbsAccountID *string
	OrgExternalID   string
}

// Context key type defined to avoid collisions in other pkgs using context
// See https://golang.org/pkg/context/#WithValue
type contextKey string

const (
	contextSegmentClient         contextKey = "segment_client"
	contextRHITUserAccountID     contextKey = "account_id"
	contextRHITUserLocale        contextKey = "user_locale"
	contextRHITOrgId             contextKey = "org_id"
	contextRequestIPkey          contextKey = "requestIP"
	contextRequestUAkey          contextKey = "requestUserAgent"
	contextServiceAccount        contextKey = "service_account"
	contextRHITUserAccountNumber contextKey = "account_number"

	redhatIBMEmail = "@(redhat.com|ibm.com)$"
)
