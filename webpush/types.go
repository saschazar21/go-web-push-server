package webpush

import (
	"crypto/ecdh"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type StringerValidator interface {
	String() string
	Validate() error
}

type recipientKeys struct {
	P256DH string `json:"p256dh" validate:"len=87"`
	Auth   string `json:"auth" validate:"len=22"`
}

type recipientSubscription struct {
	Endpoint       string         `json:"endpoint" validate:"http_url"`
	ExpirationTime *EpochMillis   `json:"expirationTime,omitempty"`
	Keys           *recipientKeys `json:"keys" validate:"required"`
}

type recipient struct {
	ClientId    string `json:"clientId" validate:"required"`
	RecipientId string `json:"id"`

	Subscription *recipientSubscription `json:"subscription" validate:"required"`
}

func (r *recipient) Validate() (err error) {
	if err = CustomValidateStruct(r); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "invalid recipient contents", err.Error())
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}

type Epoch struct {
	time.Time
}

func (e Epoch) MarshalJSON() ([]byte, error) {
	ts := e.Unix()

	return []byte(fmt.Sprintf("%d", ts)), nil
}

func (e *Epoch) UnmarshalJSON(val []byte) (err error) {
	var epoch int64
	stringified := string(val)

	if epoch, err = strconv.ParseInt(stringified, 10, 64); err != nil {
		return
	}

	(*e).Time = time.Unix(epoch, 0)

	return
}

type EpochMillis struct {
	time.Time
}

func (e EpochMillis) MarshalJSON() ([]byte, error) {
	ts := e.UnixNano() / int64(time.Millisecond)

	return []byte(fmt.Sprintf("%d", ts)), nil
}

func (e *EpochMillis) UnmarshalJSON(val []byte) (err error) {
	var epoch int64
	stringified := string(val)

	if epoch, err = strconv.ParseInt(stringified, 10, 64); err != nil {
		return
	}

	(*e).Time = time.Unix(0, epoch*int64(time.Millisecond))

	return
}

var _ sql.Scanner = (*EpochMillis)(nil)

func (e *EpochMillis) Scan(src interface{}) (err error) {
	switch src := src.(type) {
	case time.Time:
		(*e).Time = src
	case int64:
		(*e).Time = time.Unix(0, src*int64(time.Millisecond))
	default:
		err = fmt.Errorf("unsupported type: %T", src)
	}

	return
}

var _ driver.Valuer = (*EpochMillis)(nil)

func (e EpochMillis) Value() (val driver.Value, err error) {
	return e.Time, nil
}

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
	if err = CustomValidateStruct(w); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "invalid webpush parameters", err.Error())
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}
