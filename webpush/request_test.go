package webpush

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func handleRequest(res http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		res.Header().Add(k, strings.Join(v, ", "))
	}

	res.WriteHeader(200)
	res.Write([]byte("OK"))
}

func TestRequest(t *testing.T) {
	t.Setenv(VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
`)
	t.Setenv(VAPID_SUBJECT_ENV, "test@example.com")

	type test struct {
		name    string
		payload *WebPushRequest
		wantErr bool
	}

	testServer := httptest.NewServer(http.HandlerFunc(handleRequest))

	defer func() {
		testServer.Close()
	}()

	tests := []test{
		{
			"validates",
			&WebPushRequest{
				testServer.URL,
				[]byte{0x00, 0x00, 0x00, 0x00},
				&WithWebPushParams{
					"",
					3600,
					"normal",
				},
			},
			false,
		}, {
			"validates negative TTL value",
			&WebPushRequest{
				testServer.URL,
				[]byte{0x00, 0x00, 0x00, 0x00},
				&WithWebPushParams{
					"",
					-1,
					"normal",
				},
			},
			false,
		}, {
			"validates overflowing TTL value",
			&WebPushRequest{
				testServer.URL,
				[]byte{0x00, 0x00, 0x00, 0x00},
				&WithWebPushParams{
					"",
					MAX_TTL_VALUE + 2,
					"normal",
				},
			},
			false,
		}, {
			"fails to validate endpoint",
			&WebPushRequest{
				"htp://push.example",
				[]byte{0x00, 0x00, 0x00, 0x00},
				&WithWebPushParams{
					"",
					3600,
					"normal",
				},
			},
			true,
		}, {
			"fails to validate payload",
			&WebPushRequest{
				testServer.URL,
				make([]byte, 4097),
				&WithWebPushParams{
					"",
					3600,
					"normal",
				},
			},
			true,
		}, {
			"fails to validate urgency",
			&WebPushRequest{
				testServer.URL,
				make([]byte, 4096),
				&WithWebPushParams{
					"",
					3600,
					"normaly",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var res *http.Response

			if res, err = tt.payload.Send(); (err != nil) != tt.wantErr {
				t.Errorf("TestRequest err = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {
				assert.NotEmpty(t, res.Header.Get("Authorization"))
				assert.Equal(t, res.Header.Get("Content-Encoding"), "aes128gcm")
			}
		})
	}
}
