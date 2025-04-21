package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

func TestRequestDispatch(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	middleware := NewRegionProxy(
		context.Background(),
		WithGetDispatchHostFunc(func(ctx context.Context, logger logging.Logger,
			r *http.Request, connection *sdk.Connection) (string, error) {
			return "api.test.openshift.com", nil
		}),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusBadGateway))
}

func TestRequestNotDispatch(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	middleware := NewRegionProxy(
		context.Background(),
		WithGetDispatchHostFunc(func(ctx context.Context, logger logging.Logger,
			r *http.Request, connection *sdk.Connection) (string, error) {
			return "", nil
		}),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestErrorInGetDispatchHostFunc(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	middleware := NewRegionProxy(
		context.Background(),
		WithGetDispatchHostFunc(func(ctx context.Context, logger logging.Logger,
			r *http.Request, connection *sdk.Connection) (string, error) {
			return "", fmt.Errorf("errors")
		}),
	)

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
}
