package api_utils

import (
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/saschazar21/go-web-push-server/webpush"
)

var decoder = schema.NewDecoder()

type recipientParams struct {
	RecipientId string `schema:"id"`
}

func DecodeRecipientParams(r *http.Request) (params *recipientParams, err error) {
	params = new(recipientParams)

	decoder.IgnoreUnknownKeys(true)
	if err = decoder.Decode(params, r.URL.Query()); err != nil {
		log.Println(err)

		err = webpush.NewResponseError(webpush.BAD_REQUEST_ERROR, http.StatusBadRequest)
		return
	}

	return
}
