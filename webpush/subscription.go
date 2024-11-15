package webpush

import (
	"log"
	"net/http"

	"github.com/uptrace/bun"
)

type pushSubscriptionKeys struct {
	bun.BaseModel `bun:"keys"`

	P256DH string `json:"p256dh" validate:"len=87" bun:"p256dh,pk"`
	Auth   string `json:"auth" validate:"len=22" bun:"auth_secret,notnull"`
}

type pushSubscription struct {
	bun.BaseModel `bun:"subscription"`

	Endpoint       string  `json:"endpoint" validate:"http_url" bun:"endpoint,pk"`
	ExpirationTime float64 `json:"expirationTime,omitempty" bun:"expiration_time"`

	Keys pushSubscriptionKeys `json:"keys" validate:"required" bun:"rel:has-one,join:keys"`
}

type recipient struct {
	bun.BaseModel `bun:"recipient"`

	ClientId string `json:"client_id" validate:"required" bun:"client_id,pk"`
	Subject  string `json:"sub" validate:"required" bun:"id,pk"`

	Subscriptions []pushSubscription `json:"subscriptions" validate:"required,dive" bun:"rel:has-many,join:subscription"`
}

func (s *recipient) Validate() (err error) {
	if err = CustomValidateStruct(s); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "invalid recipient contents", err.Error())
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}

func ParseSubscription(r *http.Request) (sub *recipient, err error) {
	sub = new(recipient)

	if err = ParseBody(r, sub); err != nil {
		log.Println(err)

		responseErr, ok := err.(ResponseError)

		if !ok {
			payload := NewErrorResponse(http.StatusBadRequest, "failed to decode recipient")
			return sub, NewResponseError(payload, http.StatusBadRequest)
		}

		return sub, responseErr
	}

	if err = sub.Validate(); err != nil {
		return nil, err
	}

	return
}
