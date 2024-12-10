package webpush

import (
	"bytes"
	"crypto/tls"
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
	*WithSalt
	*WithPublicKey
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
		http.CanonicalHeaderKey("Authorization"):    {fmt.Sprintf("WebPush %s", jwt)},
		http.CanonicalHeaderKey("Content-Encoding"): {"aes128gcm"},
		http.CanonicalHeaderKey("Content-Type"):     {"application/octet-stream"},
		http.CanonicalHeaderKey("Crypto-Key"):       {fmt.Sprintf("dh=%s;p256ecdsa=%s", r.WithPublicKey.String(), key)},
		http.CanonicalHeaderKey("Encryption"):       {fmt.Sprintf("salt=%s", r.WithSalt.String())},
		http.CanonicalHeaderKey("TTL"):              {fmt.Sprintf("%d", r.TTL)},
		// Microsoft Edge header values: https://learn.microsoft.com/en-us/windows/apps/design/shell/tiles-and-notifications/push-request-response-headers#request-parameters
		"X-WNS-Type":         {"wns/raw"},
		"X-WNS-Cache-Policy": {"cache"},
	}

	if r.Topic != "" {
		req.Header.Add("Topic", r.Topic)
	}

	if r.Urgency != "" {
		req.Header.Add("Urgency", r.Urgency)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	if res, err = client.Do(req); err != nil {
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}
