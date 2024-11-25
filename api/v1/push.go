package v1

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

var decoder = schema.NewDecoder()

func decodeParams(r *http.Request) (params *webpush.WebPushDetails, err error) {
	params = new(webpush.WebPushDetails)

	if err = decoder.Decode(params, r.URL.Query()); err != nil {
		log.Println(err)

		err = webpush.NewResponseError(webpush.BAD_REQUEST_ERROR, http.StatusBadRequest)
		return
	}

	return
}

func deleteObsoleteSubscriptions(ctx context.Context, db *bun.DB, errorObjects []webpush.ErrorObject) (err error) {
	for _, errObj := range errorObjects {
		if errObj.Meta == nil || (errObj.Status != http.StatusGone && errObj.Status != http.StatusNotFound) {
			continue
		}

		webpush.DeleteSubscriptionByEndpoint(ctx, db, errObj.Meta.Endpoint)
	}

	return
}

func sendPushNotifications(subscriptions []webpush.PushSubscription, payload []byte, params *webpush.WithWebPushParams) (errorObjects []webpush.ErrorObject, err error) {
	var notifications []*webpush.WebPush
	var statusCode int

	for _, sub := range subscriptions {
		push := new(webpush.WebPush)

		if push, err = webpush.NewWebPush(&sub); err != nil {
			return
		}

		notifications = append(notifications, push)
	}

	for _, notification := range notifications {
		var errObj webpush.ErrorObject
		var res *http.Response

		if res, err = notification.Send(payload, params); err != nil {
			return
		}

		switch res.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusNoContent:
			continue
		case http.StatusBadRequest:
			errObj = webpush.BAD_REQUEST_ERROR.Errors[0]
			if statusCode != http.StatusInternalServerError {
				statusCode = http.StatusBadRequest
			}
		case http.StatusNotFound:
			errObj = webpush.NewErrorResponse(http.StatusNotFound, "Subscription Not Found").Errors[0]
		case http.StatusGone:
			errObj = webpush.NewErrorResponse(http.StatusGone, "Subscription Expired").Errors[0]
		case http.StatusTooManyRequests:
			errObj = webpush.NewErrorResponse(http.StatusTooManyRequests, "Too Many Requests").Errors[0]
			errObj.Detail = fmt.Sprintf("Retry after %s", res.Header.Get("Retry-After"))
		default:
			errObj = webpush.NewErrorResponse(http.StatusInternalServerError, "Internal Server Error").Errors[0]
			statusCode = http.StatusInternalServerError
		}

		errObj.Meta = &webpush.ErrorMeta{
			Endpoint: notification.Endpoint,
		}

		errorObjects = append(errorObjects, errObj)
	}

	if len(errorObjects) > 0 {
		if statusCode == 0 {
			statusCode = errorObjects[0].Status
		}

		err = webpush.NewResponseError(&webpush.ErrorResponse{Errors: errorObjects}, statusCode)
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

	// TODO: add client to params

	if err = params.Validate(); err != nil {
		webpush.WriteResponseError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	if _, err = buf.ReadFrom(r.Body); err != nil {
		log.Println(err)

		webpush.WriteResponseError(w, webpush.NewResponseError(webpush.BAD_REQUEST_ERROR, http.StatusBadRequest))
		return
	}

	if len(buf.Bytes()) == 0 {
		payload := webpush.NewErrorResponse(http.StatusBadRequest, "empty payload")
		webpush.WriteResponseError(w, webpush.NewResponseError(payload, http.StatusBadRequest))
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

	if len(subs) == 0 {
		payload := webpush.NewErrorResponse(http.StatusNotFound, "no subscriptions found")
		webpush.WriteResponseError(w, webpush.NewResponseError(payload, http.StatusNotFound))
		return
	}

	if errorObjects, err := sendPushNotifications(subs, buf.Bytes(), params.WithWebPushParams); err != nil {
		log.Println(err)

		deleteObsoleteSubscriptions(r.Context(), db, errorObjects)

		webpush.WriteResponseError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
