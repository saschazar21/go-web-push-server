package webpush

import (
	"log"
	"net/http"
)

type pushSubscriptionKeys struct {
	P256DH string `json:"p256dh" validate:"len=87"`
	Auth   string `json:"auth" validate:"len=22"`
}

type pushSubscription struct {
	Endpoint       string               `json:"endpoint" validate:"http_url"`
	ExpirationTime float64              `json:"expirationTime,omitempty"`
	Keys           pushSubscriptionKeys `json:"keys" validate:"required"`
}

type subscription struct {
	ClientId     string           `json:"client_id" validate:"required"`
	Subject      string           `json:"sub" validate:"required"`
	Subscription pushSubscription `json:"subscription" validate:"required"`
}

func (s *subscription) Validate() (err error) {
	if err = CustomValidateStruct(s); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "invalid subscription contents", err.Error())
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}

func ParseSubscription(r *http.Request) (sub *subscription, err error) {
	sub = new(subscription)

	if err = ParseBody(r, sub); err != nil {
		log.Println(err)

		responseErr, ok := err.(ResponseError)

		if !ok {
			payload := NewErrorResponse(http.StatusBadRequest, "failed to decode subscription")
			return sub, NewResponseError(payload, http.StatusBadRequest)
		}

		return sub, responseErr
	}

	if err = sub.Validate(); err != nil {
		return nil, err
	}

	return
}
