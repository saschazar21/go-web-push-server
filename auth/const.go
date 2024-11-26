package auth

import (
	"net/http"

	"github.com/saschazar21/go-web-push-server/webpush"
)

const (
	BASIC_AUTH_PASSWORD_ENV = "BASIC_AUTH_PASSWORD"
)

var (
	FORBIDDEN_ERROR = &webpush.ErrorResponse{
		Errors: []webpush.ErrorObject{
			{
				Status: http.StatusForbidden,
				Title:  "Forbidden",
			},
		},
	}

	UNAUTHORIZED_ERROR = &webpush.ErrorResponse{
		Errors: []webpush.ErrorObject{
			{
				Status: http.StatusUnauthorized,
				Title:  "Unauthorized",
			},
		},
	}
)
