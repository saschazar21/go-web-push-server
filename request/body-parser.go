package request

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/utils"
)

func ParseBody(req *http.Request, iface any) (err error) {
	if req.Method != http.MethodPost {
		headers := map[string][]string{
			http.CanonicalHeaderKey("allow"): {http.MethodPost},
		}

		return errors.NewResponseError(errors.METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, headers)
	}

	contentType := req.Header.Get("content-type")

	if contentType != utils.APPLICATION_JSON {
		payload := errors.NewErrorResponse(http.StatusBadRequest, fmt.Sprintf("content-type must equal %s", utils.APPLICATION_JSON))
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	if err = json.NewDecoder(req.Body).Decode(iface); err != nil {
		log.Println(err)

		payload := errors.NewErrorResponse(http.StatusBadRequest, "failed to parse JSON body")
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	return
}
