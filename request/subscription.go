package request

import (
	"encoding/base64"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/models"
	"github.com/saschazar21/go-web-push-server/utils"
)

func ParseSubscriptionRequest(req *http.Request) (sub *models.PushSubscription, err error) {
	r := &utils.Recipient{}

	if err = ParseBody(req, r); err != nil {
		log.Println(err)

		responseErr, ok := err.(errors.ResponseError)

		if !ok {
			payload := errors.NewErrorResponse(http.StatusBadRequest, "failed to decode recipient")
			return sub, errors.NewResponseError(payload, http.StatusBadRequest)
		}

		return sub, responseErr
	}

	if err = utils.CustomValidateStruct(r); err != nil {
		log.Printf("validation of subscription request failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "validation failed", err.Error())
		return sub, errors.NewResponseError(payload, http.StatusBadRequest)
	}

	decodedClientKey, err := base64.RawURLEncoding.DecodeString(r.Subscription.Keys.P256DH)
	if err != nil {
		log.Printf("decoding client public key failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "invalid client public key", err.Error())
		return sub, errors.NewResponseError(payload, http.StatusBadRequest)
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(r.Subscription.Keys.Auth)
	if err != nil {
		log.Printf("decoding auth secret failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "invalid auth secret", err.Error())
		return sub, errors.NewResponseError(payload, http.StatusBadRequest)
	}

	sub = &models.PushSubscription{
		Endpoint:       (*utils.EncryptedString)(&r.Subscription.Endpoint),
		ExpirationTime: r.Subscription.ExpirationTime,
		ClientId:       r.ClientId,
		RecipientId:    r.RecipientId,
		Keys: &models.SubscriptionKeys{
			P256DH:     (*utils.EncryptedBytes)(&decodedClientKey),
			AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
		},
	}

	if err = sub.Validate(); err != nil {
		return nil, err
	}

	return
}
