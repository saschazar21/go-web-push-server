package v1

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/db"
	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/request"
	"github.com/uptrace/bun"
)

func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	clientId, err := auth.HandleBasicAuth(r)

	if err != nil {
		errors.WriteResponseError(w, err)
		return
	}

	sub, err := request.ParseSubscriptionRequest(r)

	if err != nil {
		log.Println(err)

		errors.WriteResponseError(w, err)
		return
	}

	if clientId != sub.ClientId {
		errors.WriteResponseError(w, errors.NewResponseError(auth.FORBIDDEN_ERROR, http.StatusBadRequest))
		return
	}

	var conn *bun.DB
	conn, err = db.Connect()

	if err != nil {
		log.Println(err)

		errors.WriteResponseError(w, err)
		return
	}

	defer conn.Close()

	ctx := r.Context()

	var tx bun.Tx
	tx, err = conn.BeginTx(ctx, &sql.TxOptions{})

	if err != nil {
		log.Println(err)

		errors.WriteResponseError(w, err)
		return
	}

	if err = sub.Save(ctx, tx); err != nil {
		log.Println(err)

		errors.WriteResponseError(w, err)
		tx.Rollback()
		return
	}

	if err = tx.Commit(); err != nil {
		log.Println(err)

		errors.WriteResponseError(w, err)
		tx.Rollback()
		return
	}

	w.WriteHeader(http.StatusCreated)
}
