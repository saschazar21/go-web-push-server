package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saschazar21/go-web-push-server/webpush"
	"gotest.tools/v3/assert"
)

func TestHandleBasicAuth(t *testing.T) {
	type test struct {
		name       string
		username   string
		password   string
		wantStatus int
	}

	tests := []test{
		{
			"should return 401 Unauthorized on missing credentials",
			"",
			"",
			401,
		},
		{
			"should return 403 Forbidden on invalid credentials",
			"test",
			"test",
			403,
		},
		{
			"should return 401 Unauthorized on missing username",
			"",
			"123",
			401,
		},
		{
			"should return 500 Internal Server Error on unset environment variable",
			"",
			"",
			500,
		},
		{
			"should return valid credentials",
			"admin",
			"123",
			200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantStatus != 500 {
				t.Setenv(BASIC_AUTH_PASSWORD_ENV, "123")
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.username != "" && tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			clientId, err := HandleBasicAuth(req)

			if tt.wantStatus == 200 {
				assert.NilError(t, err)
				assert.Equal(t, tt.username, clientId)
			} else {
				responseErr, _ := err.(webpush.ResponseError)

				assert.Equal(t, tt.wantStatus, responseErr.StatusCode)
			}
		})
	}
}
