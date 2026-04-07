package request

import (
	"crypto/ecdh"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/utils"
)

type WithPublicKey struct {
	*ecdh.PublicKey `json:"publicKey" validate:"required"`
}

func (w *WithPublicKey) String() string {
	bytes := w.PublicKey.Bytes()

	return base64.RawURLEncoding.EncodeToString(bytes)
}

type WithSalt struct {
	Salt []byte `json:"salt" validate:"required,len=16"`
}

func (s *WithSalt) String() string {
	return base64.RawURLEncoding.EncodeToString(s.Salt)
}

type WithWebPushParams struct {
	Topic   string `json:"topic,omitempty" schema:"topic"`
	TTL     int64  `json:"ttl" schema:"ttl" validate:"gte=0"`
	Urgency string `json:"urgency,omitempty" schema:"urgency" validate:"omitempty,oneof=very-low low normal high"` // see https://datatracker.ietf.org/doc/html/rfc8030#section-5.3
}

type WebPushDetails struct {
	ClientId    string `json:"client" schema:"client" validate:"required"`
	RecipientId string `json:"id,omitempty" schema:"id"`

	*WithWebPushParams
}

func (w *WebPushDetails) Validate() (err error) {
	if err = utils.CustomValidateStruct(w); err != nil {
		log.Println(err)

		payload := errors.NewErrorResponse(http.StatusBadRequest, "invalid webpush parameters", err.Error())
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	return
}
