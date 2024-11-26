package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saschazar21/go-web-push-server/auth"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/webpush"
	"gotest.tools/v3/assert"
)

func TestHandleSubscribe(t *testing.T) {
	basicAuthPassword := "123"
	t.Setenv(auth.BASIC_AUTH_PASSWORD_ENV, basicAuthPassword)
	t.Setenv("CWD", "../../")

	type keys struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	}

	type subscription struct {
		Endpoint       string `json:"endpoint"`
		ExpirationTime int64  `json:"expirationTime,omitempty"`
		Keys           keys   `json:"keys"`
	}

	type recipient struct {
		ClientId     string       `json:"clientId"`
		RecipientId  string       `json:"id"`
		Subscription subscription `json:"subscription"`
	}

	type test struct {
		name       string
		method     string
		payload    interface{}
		wantStatus int
	}

	ctx := context.Background()

	// create test container
	c, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	tests := []test{
		{
			"should return 401 Unauthorized on invalid subscription",
			http.MethodPost,
			recipient{
				ClientId:    "",
				RecipientId: "test user",
				Subscription: subscription{
					Endpoint:       "https://example.com",
					ExpirationTime: 0,
					Keys: keys{
						P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
						Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
					},
				},
			},
			http.StatusUnauthorized,
		},
		{
			"should return 405 Method Not Allowed on invalid method",
			http.MethodPut,
			recipient{
				ClientId:    "test client",
				RecipientId: "test user",
				Subscription: subscription{
					Endpoint: "https://example.com",
					Keys: keys{
						P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
						Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
					},
				},
			},
			http.StatusMethodNotAllowed,
		},
		{
			"should return 201 Created on successful subscription",
			http.MethodPost,
			recipient{
				ClientId:    "test client",
				RecipientId: "test user",
				Subscription: subscription{
					Endpoint: "https://example.com",
					Keys: keys{
						P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
						Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
					},
				},
			},
			http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			server := new(httptest.Server)

			t.Cleanup(func() {
				server.Close()
				c.Restore(ctx)
			})

			server = httptest.NewServer(http.HandlerFunc(HandleSubscribe))

			var buf []byte
			buf, err = json.Marshal(tt.payload)

			if err != nil {
				t.Errorf("TestHandleSubscribe err = %v, wantErr = %v", err, nil)
			}

			req, _ := http.NewRequest(tt.method, server.URL, bytes.NewBuffer(buf))
			req.Header.Add("content-type", webpush.APPLICATION_JSON)

			clientId := ""
			if recipient, ok := tt.payload.(recipient); ok {
				clientId = recipient.ClientId
			}

			req.SetBasicAuth(clientId, basicAuthPassword)

			res := new(http.Response)
			res, err = http.DefaultClient.Do(req)

			if err != nil {
				t.Errorf("TestHandleSubscribe err = %v, wantErr = %v", err, nil)
			}

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}
