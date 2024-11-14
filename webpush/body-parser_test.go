package webpush

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBody(t *testing.T) {
	type member struct {
		Age  int    `json:"age"`
		Name string `json:"name"`
	}

	type test struct {
		name    string
		method  string
		header  http.Header
		payload []byte
		iface   member
		cmp     member
		wantErr bool
	}

	tests := []test{
		{
			"parses simple JSON payload",
			http.MethodPost,
			http.Header{
				http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
			},
			[]byte("{\"age\":29,\"name\":\"John\"}"),
			member{},
			member{29, "John"},
			false,
		},
		{
			"fails to parse malformatted payload",
			http.MethodPost,
			http.Header{
				http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
			},
			[]byte("\"age\":29,\"name\":\"John\"}"),
			member{},
			member{},
			true,
		},
		{
			"fails to parse PUT request",
			http.MethodPut,
			http.Header{
				http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
			},
			[]byte("{\"age\":29,\"name\":\"John\"}"),
			member{},
			member{},
			true,
		},
		{
			"fails to parse text/plain payload",
			http.MethodPost,
			http.Header{
				http.CanonicalHeaderKey("content-type"): {TEXT_PLAIN},
			},
			[]byte("{\"age\":29,\"name\":\"John\"}"),
			member{},
			member{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com", bytes.NewBuffer(tt.payload))
			req.Header = tt.header

			if err := ParseBody(req, &tt.iface); (err != nil) != tt.wantErr {
				t.Errorf("TestParseBody err = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.cmp, tt.iface)
		})
	}
}
