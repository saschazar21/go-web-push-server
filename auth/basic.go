package auth

import (
	"log"
	"net/http"
	"os"

	"github.com/saschazar21/go-web-push-server/errors"
)

func HandleBasicAuth(r *http.Request) (clientId string, err error) {
	var ok bool
	var password string

	passwordEnv := os.Getenv(BASIC_AUTH_PASSWORD_ENV)

	if passwordEnv == "" {
		log.Printf("missing environment variable %s\n", BASIC_AUTH_PASSWORD_ENV)

		err = errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
		return
	}

	clientId, password, ok = r.BasicAuth()

	if !ok || clientId == "" {
		log.Println("missing basic authentication")

		err = errors.NewResponseError(UNAUTHORIZED_ERROR, http.StatusUnauthorized, http.Header{
			http.CanonicalHeaderKey("WWW-Authenticate"): []string{"Basic realm=\"webpush\""},
		})
		return
	}

	if password != passwordEnv {
		log.Printf("invalid basic authentication: %s vs. %s\n", passwordEnv, password)

		err = errors.NewResponseError(FORBIDDEN_ERROR, http.StatusForbidden)
		return
	}

	return
}
