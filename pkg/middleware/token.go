package middleware

import (
	"context"
	"net/http"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

const (
	accessToken         = "AccessToken"
	ContextAccountIDKey = "accountID"
)

type TokenMiddleware interface {
	AuthenticateToken(next http.Handler) http.Handler
}

type TokenAuthMiddleware struct {
	connection *sdk.Connection
}

var _ TokenMiddleware = &TokenAuthMiddleware{}

func NewTokenAuthMiddleware(connection *sdk.Connection) (*TokenAuthMiddleware, error) {
	middleware := TokenAuthMiddleware{
		connection: connection,
	}

	return &middleware, nil
}

func (t *TokenAuthMiddleware) Authenticate(ctx context.Context, headers http.Header) string {
	var token string

	// parse Authorization: AccessToken header
	authHeader := headers.Get("Authorization")
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) == 2 && headerParts[0] == accessToken {
		authParts := strings.Split(headerParts[1], ":")
		if len(authParts) == 2 {
			token = authParts[1]
		}
	}

	// request Token Authorization to find account from header information
	if len(token) > 0 {
		request, err := v1.NewTokenAuthorizationRequest().AuthorizationToken(token).Build()
		if err == nil {
			api := t.connection.AccountsMgmt().V1()
			response, err := api.TokenAuthorization().Post().Request(request).Send()

			if err == nil {
				readResponse, ok := response.GetResponse()
				if ok {
					account := readResponse.Account()
					accountId := account.ID()
					return accountId
				}

			}
		}
	}
	return ""
}

func (t *TokenAuthMiddleware) AuthenticateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountId := t.Authenticate(r.Context(), r.Header)
		ctx := context.WithValue(r.Context(), ContextAccountIDKey, accountId)
		*r = *r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
