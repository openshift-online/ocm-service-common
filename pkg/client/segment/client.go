package segment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/analytics-go/v3"

	clientPkg "github.com/openshift-online/ocm-service-common/pkg/client"
	logger "github.com/openshift-online/ocm-service-common/pkg/ocmlogger"
)

type Client struct {
	httpClient *http.Client
	router     *mux.Router

	// Configuration
	Config  *ClientConfiguration
	BaseURL *url.URL

	// Services
	TrackService Service
}

var _ clientPkg.ServiceClient = &Client{}

var server *httptest.Server

var redhatIBMEmailRe *regexp.Regexp

func init() {
	redhatIBMEmailRe = regexp.MustCompile(redhatIBMEmail)
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var values map[string]string

		// fetch error from body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			ctx := r.Context()
			ulog := logger.NewOCMLogger(ctx)
			ulog.Contextual().Error(err, "could not read body")
		}
		_ = json.Unmarshal(body, &values)

		// return this error
		w.Header().Set("Content-Type", "application/json")
		code, _ := strconv.ParseInt(values["error"], 10, strconv.IntSize)
		w.WriteHeader(int(code))
	}))
}

func NewClient(ctx context.Context, config *ClientConfiguration) (*Client, error) {
	// Ensure baseURL can be parsed and has a trailing slash
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		return nil, err
	}

	client := &Client{
		Config:  config,
		BaseURL: baseURL,
	}
	client.httpClient = clientPkg.AddMetricsMiddleware(client, server.Client())
	client.TrackService = &TrackService{
		config: config,
		aConfig: analytics.Config{
			Endpoint:  config.BaseURL,
			BatchSize: 1,
			Verbose:   true,
			Logger:    logger.NewSegmentLogWrapper(),
			RetryAfter: func(attempt int) time.Duration {
				return time.Duration(attempt * 10)
			},
		},
		client: client,
	}

	return client, nil
}

func NewClientMock(ctx context.Context, config *ClientConfiguration) (*Client, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	client.TrackService = &TrackServiceMock{
		config: config,
		client: client,
	}
	return client, nil
}

func getSegmentClientFromContext(ctx context.Context) *Client {
	segmentClient, ok := ctx.Value(contextSegmentClient).(*Client)
	if ok {
		return segmentClient
	}
	return nil
}

// SetSegmentContext params:
// remoteAddr = r.RemoteAddr
// forwardedFor = r.Header.Get("X-FORWARDED-FOR")
// userAgent = r.UserAgent()
func SetSegmentContext(ctx context.Context, payload AuthPayload, segmentClient *Client, rhitWebUserId, userAgent, remoteAddr, forwardedFor string) context.Context {
	// rhitWebUserId comes from account. It is empty in case of ServiceAccount, so let's use SA's RHITUserID
	if rhitWebUserId == "" && payload.ClientID != "" {
		rhitWebUserId = payload.RHITUserID
	}
	ctx = context.WithValue(ctx, contextRHITUserAccountID, rhitWebUserId)
	ctx = context.WithValue(ctx, contextRHITUserLocale, payload.Locale)
	ctx = context.WithValue(ctx, contextRHITOrgId, payload.OrgId)

	ip := remoteAddr
	// capitalisation doesn't matter
	if forwardedFor != "" {
		// Got X-Forwarded-For
		ip = forwardedFor // If it's a single IP, then awesome!

		// If we got an array... grab the first IP
		if ips := strings.Split(forwardedFor, ","); len(ips) > 1 {
			ip = ips[0]
		}
	}
	if ipPort := strings.Split(ip, ":"); len(ipPort) > 1 {
		ip = ipPort[0]
	}
	userIP := net.ParseIP(ip)
	ctx = context.WithValue(ctx, contextRequestIPkey, userIP)
	ctx = context.WithValue(ctx, contextRequestUAkey, userAgent)

	ctx = context.WithValue(ctx, contextServiceAccount, payload.ClientID != "")
	ctx = context.WithValue(ctx, contextSegmentClient, segmentClient)
	return context.WithValue(ctx, contextRHITUserAccountNumber, payload.RHITAccountNumber)
}

func (c *Client) GetServiceName() string {
	return "segment"
}

func (c *Client) GetRouter() *mux.Router {
	if c.router == nil {
		c.router = mux.NewRouter()
		c.router.Path("/data/track").Methods(http.MethodPost)
	}
	return c.router
}

func (c *Client) getUserId(ctx context.Context, creatorRhitWebUserId *string) string {
	if creatorRhitWebUserId != nil {
		return *creatorRhitWebUserId
	}
	if userId, ok := ctx.Value(contextRHITUserAccountID).(string); ok {
		return userId
	}
	return ""
}

func (c *Client) track(retErr error) {
	var (
		request *http.Request
		err     error
	)

	u := fmt.Sprintf("%s/data/track", server.URL)
	code := "200"
	if retErr != nil {
		code = retErr.Error()
	}
	values := map[string]string{
		"error": code,
	}
	jsonData, _ := json.Marshal(values)
	if request, err = http.NewRequest(http.MethodPost, u, bytes.NewBuffer(jsonData)); err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	if _, err = c.httpClient.Do(request); err != nil {
		return
	}
}

type service struct {
	config  *ClientConfiguration
	client  *Client
	aConfig analytics.Config
}
