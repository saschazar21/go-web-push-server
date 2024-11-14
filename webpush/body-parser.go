package webpush

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func ParseBody(req *http.Request, iface any) (err error) {
	if req.Method != http.MethodPost {
		headers := map[string][]string{
			http.CanonicalHeaderKey("allow"): {http.MethodPost},
		}

		return NewResponseError(METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, headers)
	}

	contentType := req.Header.Get("content-type")

	if contentType != APPLICATION_JSON {
		payload := NewErrorResponse(http.StatusBadRequest, fmt.Sprintf("content-type must equal %s", APPLICATION_JSON))
		return NewResponseError(payload, http.StatusBadRequest)
	}

	if err = json.NewDecoder(req.Body).Decode(iface); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "failed to parse JSON body")
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}
