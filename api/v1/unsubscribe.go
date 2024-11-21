package v1

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

func HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		headers := http.Header{
			http.CanonicalHeaderKey("allow"): []string{http.MethodDelete},
		}

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, headers))
		return
	}

	// TODO: implement auth handling

	clientId := r.URL.Query().Get("client")
	recipientId := r.URL.Query().Get("id")

	if clientId == "" {
		payload := webpush.NewErrorResponse(http.StatusBadRequest, "missing client ID")
		webpush.WriteResponseError(w, webpush.NewResponseError(payload, http.StatusBadRequest))
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
