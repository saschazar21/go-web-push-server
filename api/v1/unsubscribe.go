package v1

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	api_utils "github.com/saschazar21/go-web-push-server/api/_utils"
	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

func decodeUnsubscribeRecipient(r *http.Request) (recipientId string, err error) {
	var names []string
	var values []string

	if values, names, err = api_utils.HandleURLRegex(r, "/api/v1/unsubscribe/(?P<id>[^/]+)$"); err != nil || len(values) == 0 {
		return
	}

	for i, name := range names {
		if name == "id" {
			recipientId = values[i]
			break
		}
	}

	return
}

func HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	var err error

	var recipientId string
	if recipientId, err = decodeUnsubscribeRecipient(r); err != nil {
		webpush.WriteResponseError(w, err)
		return
	}

	if recipientId == "" {
		recipientParams, err := api_utils.DecodeRecipientParams(r)

		if err != nil {
			webpush.WriteResponseError(w, err)
			return
		}

		recipientId = recipientParams.RecipientId
	}

	var clientId string
	if clientId, err = auth.HandleBasicAuth(r); err != nil {
		webpush.WriteResponseError(w, err)
		return
	}

	if r.Method != http.MethodDelete {
		headers := http.Header{
			http.CanonicalHeaderKey("allow"): []string{http.MethodDelete},
		}

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, headers))
		return
	}

	db, err := webpush.ConnectToDatabase()

	if err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	defer db.Close()

	ctx := r.Context()

	var exists bool
	if exists, err = webpush.HasExistingSubscriptionsByClient(ctx, db, clientId); err != nil || !exists {
		if err != nil {
			log.Println(err)

			webpush.WriteResponseError(w, err)
			return
		}

		payload := webpush.NewErrorResponse(http.StatusNotFound, fmt.Sprintf("no subscriptions found for client ID %s", clientId))

		webpush.WriteResponseError(w, webpush.NewResponseError(payload, http.StatusNotFound))
		return
	}

	var tx bun.Tx
	tx, err = db.BeginTx(ctx, &sql.TxOptions{})

	if err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	if recipientId == "" {
		if err = webpush.DeleteSubscriptionsByClient(ctx, tx, clientId); err != nil {
			log.Println(err)

			webpush.WriteResponseError(w, err)
			tx.Rollback()
			return
		}

		tx.Commit()
		w.WriteHeader(http.StatusNoContent)

		return
	}

	if err = webpush.DeleteSubscriptionsByClientAndRecipient(ctx, tx, clientId, recipientId); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		tx.Rollback()
		return
	}

	tx.Commit()
	w.WriteHeader(http.StatusNoContent)
}
