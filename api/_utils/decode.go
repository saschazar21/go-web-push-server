package api_utils

import (
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/saschazar21/go-web-push-server/errors"
)

var decoder = schema.NewDecoder()

type recipientParams struct {
	RecipientId string `schema:"id"`
}

func DecodeRecipientParams(r *http.Request) (params *recipientParams, err error) {
	params = &recipientParams{}

	decoder.IgnoreUnknownKeys(true)
	if err = decoder.Decode(params, r.URL.Query()); err != nil {
		log.Println(err)

		err = errors.NewResponseError(errors.BAD_REQUEST_ERROR, http.StatusBadRequest)
		return
	}

	return
}
