package v1

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	clientId, err := auth.HandleBasicAuth(r)

	if err != nil {
		webpush.WriteResponseError(w, err)
		return
	}

	sub, err := webpush.ParseSubscription(r)

	if err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	if clientId != sub.ClientId {
		webpush.WriteResponseError(w, webpush.NewResponseError(auth.FORBIDDEN_ERROR, http.StatusBadRequest))
		return
	}

	var db *bun.DB
	db, err = webpush.ConnectToDatabase()

	if err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	defer db.Close()

	ctx := r.Context()

	var tx bun.Tx
	tx, err = db.BeginTx(ctx, &sql.TxOptions{})

	if err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	if err = sub.Save(ctx, tx); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		tx.Rollback()
		return
	}

	if err = tx.Commit(); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		tx.Rollback()
		return
	}

	w.WriteHeader(http.StatusCreated)
}
