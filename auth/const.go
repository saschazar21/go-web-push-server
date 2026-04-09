package auth

import (
	"net/http"

	"github.com/saschazar21/go-web-push-server/errors"
)

const (
	BASIC_AUTH_PASSWORD_ENV = "BASIC_AUTH_PASSWORD"
)

var (
	FORBIDDEN_ERROR = &errors.ErrorResponse{
		Errors: []errors.ErrorObject{
			{
				Status: http.StatusForbidden,
				Title:  "Forbidden",
			},
		},
	}

	UNAUTHORIZED_ERROR = &errors.ErrorResponse{
		Errors: []errors.ErrorObject{
			{
				Status: http.StatusUnauthorized,
				Title:  "Unauthorized",
			},
		},
	}
)
