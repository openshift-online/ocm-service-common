package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/ocm/consts"
)

var _ = Describe("Deprecation Middleware", func() {
	var (
		nextHandler      http.Handler
		handler          http.Handler
		responseRecorder *httptest.ResponseRecorder
		nextCalled       bool
	)

	BeforeEach(func() {
		nextCalled = false
		nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		responseRecorder = httptest.NewRecorder()
	})

	Context("when endpoint is not deprecated", func() {
		It("should call the next handler without adding headers", func() {
			deprecatedEndpoints := map[string]DeprecatedEndpoint{}
			cfg := MiddlewareConfig{Endpoints: deprecatedEndpoints}
			handler = NewDeprecationMiddleware(cfg)(nextHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			handler.ServeHTTP(responseRecorder, req)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			Expect(nextCalled).To(BeTrue())
			Expect(responseRecorder.Header().Get(consts.DeprecationHeader)).To(BeEmpty())
			Expect(responseRecorder.Header().Get(consts.OcmDeprecationMessage)).To(BeEmpty())
		})
	})

	Context("when endpoint is deprecated but not expired", func() {
		It("should add deprecation headers and call the next handler", func() {
			sunsetDate := time.Now().Add(24 * time.Hour)
			deprecatedEndpoints := map[string]DeprecatedEndpoint{
				"/api/test": {
					Message:    "This is deprecated",
					SunsetDate: sunsetDate,
				},
			}
			cfg := MiddlewareConfig{Endpoints: deprecatedEndpoints}
			handler = NewDeprecationMiddleware(cfg)(nextHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			handler.ServeHTTP(responseRecorder, req)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			Expect(nextCalled).To(BeTrue())
			Expect(responseRecorder.Header().Get(consts.DeprecationHeader)).To(Equal(sunsetDate.Format(time.RFC3339)))
			Expect(responseRecorder.Header().Get(consts.OcmDeprecationMessage)).To(Equal("This is deprecated"))
		})
	})

	Context("when endpoint is expired", func() {
		It("should return 410 Gone and not call the next handler", func() {
			sunsetDate := time.Now().Add(-24 * time.Hour) // Expired
			deprecatedEndpoints := map[string]DeprecatedEndpoint{
				"/api/test": {
					Message:    "This is gone",
					SunsetDate: sunsetDate,
				},
			}

			var sentError *Error
			var createErrorCalled bool
			cfg := MiddlewareConfig{
				Endpoints: deprecatedEndpoints,
				CreateError: func(r *http.Request, format string, a any) Error {
					createErrorCalled = true
					return Error{
						ID:   "410",
						Code: "CLUSTERS-MGMT-410",
						Reason: fmt.Sprintf(
							"The requested resource '%s' is no longer available and will not be available again",
							r.URL.Path),
						Timestamp: time.Now().UTC(),
					}
				},
				SendError: func(w http.ResponseWriter, r *http.Request, err *Error) {
					sentError = err
					status, conversionErr := strconv.Atoi(err.ID)
					Expect(conversionErr).ToNot(HaveOccurred())
					w.WriteHeader(status)
					w.Header().Set("Content-Type", "application/json")
				},
			}

			handler = NewDeprecationMiddleware(cfg)(nextHandler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			handler.ServeHTTP(responseRecorder, req)

			Expect(responseRecorder.Code).To(Equal(http.StatusGone))
			Expect(nextCalled).To(BeFalse())
			Expect(createErrorCalled).To(BeTrue())
			Expect(sentError).ToNot(BeNil())
			Expect(sentError.Reason).To(ContainSubstring("no longer available"))
		})
	})

	Context("when endpoint with path parameter is deprecated", func() {
		It("should match the pattern and add deprecation headers", func() {
			sunsetDate := time.Now().Add(24 * time.Hour)
			deprecatedEndpoints := map[string]DeprecatedEndpoint{
				"/api/clusters/{id}": {
					Message:    "Use v2 instead",
					SunsetDate: sunsetDate,
				},
			}
			cfg := MiddlewareConfig{Endpoints: deprecatedEndpoints}
			handler = NewDeprecationMiddleware(cfg)(nextHandler)

			req := httptest.NewRequest("GET", "/api/clusters/12345", nil)
			handler.ServeHTTP(responseRecorder, req)

			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			Expect(nextCalled).To(BeTrue())
			Expect(responseRecorder.Header().Get(consts.DeprecationHeader)).To(Equal(sunsetDate.Format(time.RFC3339)))
			Expect(responseRecorder.Header().Get(consts.OcmDeprecationMessage)).To(Equal("Use v2 instead"))
		})
	})
})

var _ = Describe("matchesPattern", func() {
	type testCase struct {
		path    string
		pattern string
		matches bool
	}

	DescribeTable("path matching",
		func(tc testCase) {
			Expect(matchesPattern(tc.path, tc.pattern)).To(Equal(tc.matches))
		},
		Entry("should match identical paths", testCase{
			path:    "/api/v1/test",
			pattern: "/api/v1/test",
			matches: true,
		}),
		Entry("should match with path parameter", testCase{
			path:    "/api/v1/clusters/123",
			pattern: "/api/v1/clusters/{id}",
			matches: true,
		}),
		Entry("should not match different paths", testCase{
			path:    "/api/v1/foo",
			pattern: "/api/v1/bar",
			matches: false,
		}),
		Entry("should not match if lengths are different", testCase{
			path:    "/api/v1/clusters/123/nodes",
			pattern: "/api/v1/clusters/{id}",
			matches: false,
		}),
		Entry("should handle multiple path parameters", testCase{
			path:    "/api/v1/clusters/123/nodes/456",
			pattern: "/api/v1/clusters/{cluster_id}/nodes/{node_id}",
			matches: true,
		}),
		Entry("should handle trailing slashes", testCase{
			path:    "/api/v1/test/",
			pattern: "/api/v1/test",
			matches: true,
		}),
	)
})
