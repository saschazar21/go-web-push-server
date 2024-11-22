package v1

import (
	"bytes"
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

var decoder = schema.NewDecoder()

func decodeParams(r *http.Request) (params *webpush.WebPushDetails, err error) {
	if err = decoder.Decode(&params, r.URL.Query()); err != nil {
		log.Println(err)

		err = webpush.NewResponseError(webpush.BAD_REQUEST_ERROR, http.StatusBadRequest)
		return
	}

	if err = params.Validate(); err != nil {
		return
	}

	return
}

func sendPushNotifications(subscriptions []webpush.PushSubscription, payload []byte, params *webpush.WithWebPushParams) (err error) {
	var notifications []*webpush.WebPush

	for _, sub := range subscriptions {
		push := new(webpush.WebPush)

		if push, err = webpush.NewWebPush(&sub); err != nil {
			return
		}

		notifications = append(notifications, push)
	}

	for _, notification := range notifications {
		var res *http.Response

		if res, err = notification.Send(payload, params); err != nil {
			return
		}

		if res.StatusCode != http.StatusCreated {
			err = webpush.NewResponseError(webpush.INTERNAL_SERVER_ERROR, res.StatusCode)
			return
		}
	}

	return
}

func HandlePush(w http.ResponseWriter, r *http.Request) {
	// TODO: implement auth handling

	var err error

	if r.Method != http.MethodPost {
		header := http.Header{
			http.CanonicalHeaderKey("allow"): []string{http.MethodPost},
		}

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, header))
		return
	}

	contentType := r.Header.Get("Content-Type")

	if contentType != webpush.APPLICATION_JSON && contentType != webpush.TEXT_PLAIN {
		header := http.Header{
			http.CanonicalHeaderKey("accept-post"): []string{webpush.APPLICATION_JSON, webpush.TEXT_PLAIN},
		}

		payload := webpush.NewErrorResponse(http.StatusUnsupportedMediaType, "unsupported media type")

		webpush.WriteResponseError(w, webpush.NewResponseError(payload, http.StatusUnsupportedMediaType, header))
		return
	}

	params := new(webpush.WebPushDetails)
	if params, err = decodeParams(r); err != nil {
		webpush.WriteResponseError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	if _, err = buf.ReadFrom(r.Body); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.BAD_REQUEST_ERROR, http.StatusBadRequest))
		return
	}

	var db *bun.DB
	if db, err = webpush.ConnectToDatabase(); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.INTERNAL_SERVER_ERROR, http.StatusInternalServerError))
		return
	}

	defer db.Close()

	var subs []webpush.PushSubscription
	if params.RecipientId != "" {
		subs, err = webpush.GetSubscriptionsByClientAndRecipient(r.Context(), db, params.ClientId, params.RecipientId)

		if err != nil {
			log.Println(err)

			webpush.WriteResponseError(w, err)
			return
		}
	} else {
		subs, err = webpush.GetSubscriptionsByClient(r.Context(), db, params.ClientId)

		if err != nil {
			log.Println(err)

			webpush.WriteResponseError(w, err)
			return
		}
	}

	if err = sendPushNotifications(subs, buf.Bytes(), params.WithWebPushParams); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
