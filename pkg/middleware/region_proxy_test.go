package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

const (
	clusterId          = "mock-cluster-id"
	clusterId1         = "mock-cluster-id-1"
	clusterExternalId  = "mock-cluster-external-id"
	clusterExternalId1 = "mock-cluster-external-id-1"

	APSoutEast1 = "ap-southeast-1"
	USEast1     = "us-east-1"
	Global      = "global"

	APRhRegionId = "aws.ap-southeast-1"
)

var (
	calledNext, callDispatched bool
	// mock next http handler
	nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledNext = true
	})
	// Mock regional server to be dispatched
	mockRegionalServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callDispatched = true
	}))
	// Mock Service account for SDK connection
	saToken = getServiceAccountJwtString()
	// Mock SSO response for service account token
	mockSSO = httptest.NewServer(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, saToken)),
	)
	// Mock AMS Server
	mockAMSServer = MakeTCPServer()
	// Mock SDK connection
	connection, _ = sdk.NewConnectionBuilder().
			TokenURL(mockSSO.URL).
			URL(mockAMSServer.URL()).
			Client("foo", "bar").
			Build()
)

func TestRequestNotDispatchIfFoundInLocal(t *testing.T) {
	RegisterTestingT(t)
	calledNext = false
	callDispatched = false
	middleware := NewRegionProxy(
		context.Background(),
		WithSDKConnection(connection),
		WithGetClusterIdsHandler(mockGetClusterIdsHandler(clusterId, clusterExternalId)),
		WithCheckLocalHandler(mockCheckLocalHandler(true)),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(true))
	Expect(callDispatched).To(Equal(false))
}

func TestRequestNotDispatchIfInGlobalServer(t *testing.T) {
	RegisterTestingT(t)
	calledNext = false
	callDispatched = false
	middleware := NewRegionProxy(
		context.Background(),
		WithSDKConnection(connection),
		WithGetClusterIdsHandler(mockGetClusterIdsHandler(clusterId, clusterExternalId)),
		WithCheckLocalHandler(mockCheckLocalHandler(false)),
	)
	mockAMSServer.AppendHandlers(
		mockResponseFromAMS(true, ""),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(true))
	Expect(callDispatched).To(Equal(false))
}

func TestRequestDispatchIfInRegionServer(t *testing.T) {
	RegisterTestingT(t)
	calledNext = false
	callDispatched = false
	middleware := NewRegionProxy(
		context.Background(),
		WithSDKConnection(connection),
		WithGetClusterIdsHandler(mockGetClusterIdsHandler(clusterId, clusterExternalId)),
		WithCheckLocalHandler(mockCheckLocalHandler(false)),
		WithDispatchHandler(mockDispatchFunc),
	)
	mockAMSServer.AppendHandlers(
		mockResponseFromAMS(true, APRhRegionId),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(false))
	Expect(callDispatched).To(Equal(true))
}

func TestClusterCache(t *testing.T) {
	RegisterTestingT(t)
	calledNext = false
	callDispatched = false
	middleware := NewRegionProxy(
		context.Background(),
		WithSDKConnection(connection),
		WithGetClusterIdsHandler(mockGetClusterIdsHandler(clusterId, clusterExternalId)),
		WithCheckLocalHandler(mockCheckLocalHandler(false)),
		WithDispatchHandler(mockDispatchFunc),
	)
	mockAMSServer.AppendHandlers(
		mockResponseFromAMS(true, APRhRegionId),
		mockResponseFromAMS(false, ""),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	// first request, will access AMS to get cluster region info
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(false))
	Expect(callDispatched).To(Equal(true))

	// second request on the same cluster, get cluster region from local cache, no need to access AMS
	calledNext = false
	callDispatched = false
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(false))
	Expect(callDispatched).To(Equal(true))

	// third request to get another cluster, need to access AMS again
	middleware.getClusterIdsHandler = mockGetClusterIdsHandler(clusterId1, clusterExternalId1)
	calledNext = false
	callDispatched = false
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(calledNext).To(Equal(true))
	Expect(callDispatched).To(Equal(false))
}

func TestErrorInHandler(t *testing.T) {
	RegisterTestingT(t)
	calledNext = false
	callDispatched = false
	errMsg := "mock error"
	middleware := NewRegionProxy(
		context.Background(),
		WithSDKConnection(connection),
		WithGetClusterIdsHandler(mockGetClusterIdsHandler(clusterId, clusterExternalId)),
		WithCheckLocalHandler(func(ctx context.Context, logger logging.Logger,
			ids ClusterIds) (bool, error) {
			return false, errors.New(errMsg)
		}),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
	bodyBytes, err := io.ReadAll(recorder.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(bodyBytes)).To(Equal(fmt.Sprintf("Error in region proxy: %s", errMsg)))
}

func getServiceAccountJwtString() string {
	claims := jwt.MapClaims{
		"iss":                "sso.redhat.com",
		"typ":                "Bearer",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(1 * time.Hour).Unix(),
		"client_id":          "service-account-xyz",
		"preferred_username": "service-account-xyz",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte("secret"))
	if err != nil {
		return ""
	}
	return signedToken
}

func mockGetClusterIdsHandler(id, externalId string) getClusterIdsHandlerFunc {
	return func(ctx context.Context, logger logging.Logger,
		r *http.Request) ClusterIds {
		return ClusterIds{
			Id:         id,
			ExternalId: externalId,
		}
	}
}

func mockCheckLocalHandler(found bool) checkLocalHandlerFunc {
	return func(ctx context.Context, logger logging.Logger,
		ids ClusterIds) (bool, error) {
		return found, nil
	}
}

func mockResponseFromAMS(findFlag bool, rhRegionId string) http.HandlerFunc {
	if findFlag {
		return RespondWithJSON(http.StatusOK, fmt.Sprintf(`{
			"items": [
				{
				"cluster_id":"%s",
				"external_cluster_id":"%s",
				"kind":"Subscription",
				"rh_region_id":"%s"
				}
			],
			"kind":"SubscriptionList",
			"page":1,
			"size":1,
			"total":1
			}`, clusterId, clusterExternalId, rhRegionId))
	}
	return RespondWithJSON(http.StatusOK, `{
		"items": [
		],
		"kind":"SubscriptionList",
		"page":1,
		"size":0,
		"total":0
		}`)
}

func mockDispatchFunc(ctx context.Context, logger logging.Logger, w http.ResponseWriter, r *http.Request,
	next http.Handler, rhRegionId string) error {
	if rhRegionId != "" {
		dispatchURL, _ := url.Parse(mockRegionalServer.URL)
		r.Host = dispatchURL.Host
		proxy := httputil.NewSingleHostReverseProxy(dispatchURL)
		proxy.ServeHTTP(w, r)
	} else {
		next.ServeHTTP(w, r)
	}
	return nil
}
