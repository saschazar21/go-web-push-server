package webpush

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

const (
	MAX_TTL_VALUE = 2147483648 // see https://datatracker.ietf.org/doc/html/rfc8030#section-5.2
)

type WebPushRequest struct {
	Endpoint string `validate:"http_url"`
	Payload  []byte `validate:"required,lte=4096"`

	*WithWebPushParams
}

func (r *WebPushRequest) getOrigin() string {
	u, _ := url.Parse(r.Endpoint)

	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

func (r *WebPushRequest) String() string {
	return fmt.Sprintf("[POST HTTP/1.1]: %s", r.Endpoint)
}

func (r *WebPushRequest) Validate() error {
	if r.TTL < 0 || r.TTL > MAX_TTL_VALUE {
		r.TTL = MAX_TTL_VALUE
	}

	return CustomValidateStruct(r)
}

func (r *WebPushRequest) Send() (res *http.Response, err error) {
	if err = r.Validate(); err != nil {
		log.Printf("%s: %v\n", r.String(), err)

		return res, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var req *http.Request

	if req, err = http.NewRequest(http.MethodPost, r.Endpoint, bytes.NewBuffer(r.Payload)); err != nil {
		return res, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var jwt, key string

	if jwt, key, err = NewVAPID(r.getOrigin()); err != nil {
		log.Println(err)

		return res, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	req.Header = http.Header{
		http.CanonicalHeaderKey("Authorization"):    {fmt.Sprintf("vapid t=%s", jwt), fmt.Sprintf("k=%s", key)},
		http.CanonicalHeaderKey("Content-Encoding"): {"aes128gcm"},
		http.CanonicalHeaderKey("TTL"):              {fmt.Sprintf("%d", r.TTL)},
	}

	if r.Topic != "" {
		req.Header.Add("Topic", r.Topic)
	}

	if r.Urgency != "" {
		req.Header.Add("Urgency", r.Urgency)
	}

	if res, err = http.DefaultClient.Do(req); err != nil {
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}
