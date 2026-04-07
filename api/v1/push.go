package v1

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/schema"
	api_utils "github.com/saschazar21/go-web-push-server/api/_utils"
	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/db"
	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/models"
	"github.com/saschazar21/go-web-push-server/request"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
)

var decoder = schema.NewDecoder()

func decodePushParams(r *http.Request) (params *request.WebPushDetails, err error) {
	params = &request.WebPushDetails{}

	decoder.IgnoreUnknownKeys(true)
	if err = decoder.Decode(params, r.URL.Query()); err != nil {
		log.Println(err)

		err = errors.NewResponseError(errors.BAD_REQUEST_ERROR, http.StatusBadRequest)
		return
	}

	return
}

func decodePushRecipient(r *http.Request) (recipientId string, err error) {
	var names []string
	var values []string

	if values, names, err = api_utils.HandleURLRegex(r, "/api/v1/push/(?P<id>[^/]+)$"); err != nil || len(values) == 0 {
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

func deleteObsoleteSubscriptions(ctx context.Context, db *bun.DB, errorObjects []errors.ErrorObject) (err error) {
	for _, errObj := range errorObjects {
		if errObj.Meta == nil || (errObj.Status != http.StatusGone && errObj.Status != http.StatusNotFound) {
			continue
		}

		models.DeleteSubscriptionByEndpoint(ctx, db, errObj.Meta.Endpoint)
	}

	return
}

func sendPushNotifications(subscriptions []*models.PushSubscription, payload []byte, params *request.WithWebPushParams) (errorObjects []errors.ErrorObject, err error) {
	var notifications []*webpush.WebPush
	var statusCode int

	for _, sub := range subscriptions {
		push := new(webpush.WebPush)

		if push, err = webpush.NewWebPush(sub); err != nil {
			return
		}

		notifications = append(notifications, push)
	}

	for i, notification := range notifications {
		var errObj errors.ErrorObject
		var res *http.Response

		log.Printf("sending push notification to recipient: %s of client: %s\n", subscriptions[i].RecipientId, subscriptions[i].ClientId)

		if res, err = notification.Send(payload, params); err != nil {
			return
		}

		switch res.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusNoContent:
			continue
		case http.StatusBadRequest:
			errObj = errors.BAD_REQUEST_ERROR.Errors[0]
			if statusCode != http.StatusInternalServerError {
				statusCode = http.StatusBadRequest
			}
		case http.StatusNotFound:
			errObj = errors.NewErrorResponse(http.StatusNotFound, "subscription not found").Errors[0]
		case http.StatusGone:
			errObj = errors.NewErrorResponse(http.StatusGone, "subscription expired").Errors[0]
		case http.StatusTooManyRequests:
			errObj = errors.NewErrorResponse(http.StatusTooManyRequests, "too many requests").Errors[0]
			errObj.Detail = fmt.Sprintf("Retry after %s", res.Header.Get("Retry-After"))
		default:
			errObj = errors.NewErrorResponse(http.StatusInternalServerError, "Internal Server Error").Errors[0]
			statusCode = http.StatusInternalServerError
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)

		log.Printf("[HTTP %d]: push notification for recipient: %s failed. Reason:\n%s\n", res.StatusCode, subscriptions[i].RecipientId, buf.String())
		log.Println(res.Header)

		if errObj.Detail == "" {
			errObj.Detail = buf.String()
		}

		errObj.Meta = &errors.ErrorMeta{
			Endpoint: notification.Endpoint,
		}

		errorObjects = append(errorObjects, errObj)
	}

	if len(errorObjects) > 0 {
		if statusCode == 0 {
			statusCode = errorObjects[0].Status
		}

		err = errors.NewResponseError(&errors.ErrorResponse{Errors: errorObjects}, statusCode)
	}

	return
}

func HandlePush(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	ctx := r.Context()
	var err error

	var recipientId string
	if recipientId, err = decodePushRecipient(r); err != nil {
		errors.WriteResponseError(w, err)
		return
	}

	var clientId string
	if clientId, err = auth.HandleBasicAuth(r); err != nil {
		errors.WriteResponseError(w, err)
		return
	}

	if r.Method != http.MethodPost {
		header := http.Header{
			http.CanonicalHeaderKey("allow"): []string{http.MethodPost},
		}

		errors.WriteResponseError(w, errors.NewResponseError(errors.METHOD_NOT_ALLOWED_ERROR, http.StatusMethodNotAllowed, header))
		return
	}

	contentType := r.Header.Get("Content-Type")

	if contentType != utils.APPLICATION_JSON && contentType != utils.TEXT_PLAIN {
		header := http.Header{
			http.CanonicalHeaderKey("accept-post"): []string{utils.APPLICATION_JSON, utils.TEXT_PLAIN},
		}

		payload := errors.NewErrorResponse(http.StatusUnsupportedMediaType, "unsupported media type")

		errors.WriteResponseError(w, errors.NewResponseError(payload, http.StatusUnsupportedMediaType, header))
		return
	}

	var params *request.WebPushDetails
	if params, err = decodePushParams(r); err != nil {
		errors.WriteResponseError(w, err)
		return
	}

	params.ClientId = clientId

	if recipientId != "" {
		params.RecipientId = recipientId
	}

	if err = params.Validate(); err != nil {
		errors.WriteResponseError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	if _, err = buf.ReadFrom(r.Body); err != nil {
		log.Println(err)

		errors.WriteResponseError(w, errors.NewResponseError(errors.BAD_REQUEST_ERROR, http.StatusBadRequest))
		return
	}

	if len(buf.Bytes()) == 0 {
		payload := errors.NewErrorResponse(http.StatusBadRequest, "empty payload")
		errors.WriteResponseError(w, errors.NewResponseError(payload, http.StatusBadRequest))
		return
	}

	var conn *bun.DB
	if conn, err = db.Connect(); err != nil {
		log.Println(err)

		errors.WriteResponseError(w, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError))
		return
	}

	defer conn.Close()

	var subs []*models.PushSubscription
	if params.RecipientId != "" {
		subs, err = models.GetSubscriptionsByClientIdAndRecipientId(ctx, conn, params.ClientId, params.RecipientId)

		if err != nil {
			log.Println(err)

			errors.WriteResponseError(w, err)
			return
		}
	} else {
		subs, err = models.GetSubscriptionsByClientId(ctx, conn, params.ClientId)

		if err != nil {
			log.Println(err)

			errors.WriteResponseError(w, err)
			return
		}
	}

	if len(subs) == 0 {
		payload := errors.NewErrorResponse(http.StatusNotFound, "no subscriptions found")
		errors.WriteResponseError(w, errors.NewResponseError(payload, http.StatusNotFound))
		return
	}

	if errorObjects, err := sendPushNotifications(subs, buf.Bytes(), params.WithWebPushParams); err != nil {
		log.Println(err)

		deleteObsoleteSubscriptions(ctx, conn, errorObjects)

		errors.WriteResponseError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
