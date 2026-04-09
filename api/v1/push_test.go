package v1

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/db"
	"github.com/saschazar21/go-web-push-server/models"
	"github.com/saschazar21/go-web-push-server/request"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"gotest.tools/v3/assert"
)

func TestHandlePush(t *testing.T) {
	basicAuthPassword := "123"
	t.Setenv(auth.BASIC_AUTH_PASSWORD_ENV, basicAuthPassword)
	t.Setenv("CWD", "../../")
	t.Setenv(utils.VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(utils.VAPID_PRIVATE_KEY_ENV, `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIEpu5SUVppsnLW/X1f6Mv8h8LES1g+O/gLavQhqn4oa6oAoGCCqGSM49
AwEHoUQDQgAE06wJJOQ3HWq9+MoyF4THhhV83ca/GdmkQ562OfZiisuu6/latYaX
8gYZEShGYkSTaQx4a1Xjp6EZ/khPLHcuvQ==
-----END EC PRIVATE KEY-----	
`)
	t.Setenv(utils.VAPID_SUBJECT_ENV, "test@example.com")

	type keys struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	}

	type subscription struct {
		ClientId       string `json:"clientId"`
		RecipientId    string `json:"id"`
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
		name          string
		method        string
		contentType   string
		params        *request.WebPushDetails
		payload       []byte
		triggerStatus int
		wantStatus    int
	}

	ctx := context.Background()

	c, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestHandlePush err = %v, wantErr = %v", err, nil)
	}

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	tests := []test{
		{
			"should return 401 Unauthorized on missing client ID",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte(""),
			201,
			401,
		},
		{
			"should return 400 Bad Request on missing body",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte(""),
			201,
			400,
		},
		{
			"should return 413 Request Entity Too Large on body that is too large",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			make([]byte, 4097),
			201,
			http.StatusRequestEntityTooLarge,
		},
		{
			"should return 405 Method Not Allowed on invalid method",
			http.MethodGet,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			201,
			405,
		},
		{
			"should return 415 Unsupported Media Type on invalid content type",
			http.MethodPost,
			"text/html",
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("<b>test</b>"),
			201,
			http.StatusUnsupportedMediaType,
		},
		{
			"should return 404 Not Found on missing subscription",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "missing user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			201,
			404,
		},
		{
			"triggers 400 Bad Request",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			400,
			400,
		},
		{
			"triggers 404 Not Found",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			404,
			404,
		},
		{
			"triggers 410 Gone",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			410,
			410,
		},
		{
			"triggers 429 Gone",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			http.StatusTooManyRequests,
			http.StatusTooManyRequests,
		},
		{
			"returns 500 Internal Server Error on unknown status code",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			401,
			http.StatusInternalServerError,
		},
		{
			"should return 201 for successful push messages by client & recipient ID",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			201,
			201,
		},
		{
			"should return 201 for successful push messages by client ID",
			http.MethodPost,
			utils.TEXT_PLAIN,
			&request.WebPushDetails{
				ClientId: "test client",
				WithWebPushParams: &request.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte("test"),
			201,
			201,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conn *bun.DB
			var err error
			var applicationServer *httptest.Server
			var pushServer *httptest.Server
			var u *url.URL

			t.Cleanup(func() {
				conn.Close()
				pushServer.Close()
				c.Restore(ctx)
			})

			pushServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range r.Header {
					w.Header().Set(k, v[0])
				}
				w.WriteHeader(tt.triggerStatus)
			}))

			// create test recipient
			rec := recipient{
				Subscription: subscription{
					ClientId:    "test client",
					RecipientId: "test user",
					Endpoint:    pushServer.URL,
					Keys: keys{
						P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
						Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
					},
				},
			}

			decodedClientKey, err := base64.RawURLEncoding.DecodeString(rec.Subscription.Keys.P256DH)
			if err != nil {
				t.Fatalf("failed to decode client key: %v", err)
			}

			decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(rec.Subscription.Keys.Auth)
			if err != nil {
				t.Fatalf("failed to decode auth secret: %v", err)
			}

			sub := &models.PushSubscription{
				ClientId:    rec.Subscription.ClientId,
				RecipientId: rec.Subscription.RecipientId,
				Endpoint:    (*utils.EncryptedString)(&rec.Subscription.Endpoint),
				Keys: &models.SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedClientKey),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			}

			conn, err = db.Connect()

			if err != nil {
				t.Fatalf("TestHandlePush err = %v, wantErr = %v", err, nil)
			}

			conn.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

			if err != nil {
				t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
			}

			if _, err := conn.NewInsert().Model(sub).Exec(ctx); err != nil {
				t.Fatalf("TestHandlePush err = %v, wantErr = %v", err, nil)
			}

			if sub.Keys != nil {
				sub.Keys.PushSubscriptionHash = sub.Hash
				if _, err := conn.NewInsert().Model(sub.Keys).Exec(ctx); err != nil {
					t.Fatalf("TestHandlePush err = %v, wantErr = %v", err, nil)
				}
			}

			applicationServer = httptest.NewServer(http.HandlerFunc(HandlePush))

			if u, err = url.Parse(applicationServer.URL); err != nil {
				t.Fatalf("HandlePush err = %v, wantErr = %v", err, nil)
			}

			query := u.Query()

			query.Add("id", tt.params.RecipientId)

			if tt.params.WithWebPushParams == nil {
				tt.params.WithWebPushParams = &request.WithWebPushParams{}
			}

			// TODO: remove, once auth is implemented
			if tt.params.ClientId != "" {
				query.Add("client", tt.params.ClientId)
			}

			if tt.params.TTL > 0 {
				query.Add("ttl", strconv.FormatInt(tt.params.TTL, 10))
			}

			if tt.params.Topic != "" {
				query.Add("topic", tt.params.Topic)
			}

			if tt.params.Urgency != "" {
				query.Add("urgency", tt.params.Urgency)
			}

			u.RawQuery = query.Encode()

			log.Printf("URL: %s", u.String())

			req, err := http.NewRequest(tt.method, u.String(), bytes.NewReader(tt.payload))

			if err != nil {
				t.Fatalf("HandlePush err = %v, wantErr = %v", err, nil)
			}

			req.Header.Add("content-type", tt.contentType)
			req.SetBasicAuth(tt.params.ClientId, basicAuthPassword)

			res, err := applicationServer.Client().Do(req)

			if err != nil {
				t.Fatalf("HandlePush err = %v, wantErr = %v", err, nil)
			}

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}
