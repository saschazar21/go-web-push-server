package v1

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"gotest.tools/v3/assert"
)

func TestHandlePush(t *testing.T) {
	t.Setenv("CWD", "../../")
	t.Setenv(webpush.VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(webpush.VAPID_PRIVATE_KEY_ENV, `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIEpu5SUVppsnLW/X1f6Mv8h8LES1g+O/gLavQhqn4oa6oAoGCCqGSM49
AwEHoUQDQgAE06wJJOQ3HWq9+MoyF4THhhV83ca/GdmkQ562OfZiisuu6/latYaX
8gYZEShGYkSTaQx4a1Xjp6EZ/khPLHcuvQ==
-----END EC PRIVATE KEY-----	
`)
	t.Setenv(webpush.VAPID_SUBJECT_ENV, "test@example.com")

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
		params        *webpush.WebPushDetails
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
			"should return 400 Bad Request on invalid push request",
			http.MethodPost,
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{},
			[]byte(""),
			201,
			400,
		},
		{
			"should return 400 Bad Request on missing client ID",
			http.MethodPost,
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
					TTL: 0,
				},
			},
			[]byte(""),
			201,
			400,
		},
		{
			"should return 400 Bad Request on missing body",
			http.MethodPost,
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "missing user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId:    "test client",
				RecipientId: "test user",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			webpush.TEXT_PLAIN,
			&webpush.WebPushDetails{
				ClientId: "test client",
				WithWebPushParams: &webpush.WithWebPushParams{
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
			var db *bun.DB
			var err error
			var applicationServer *httptest.Server
			var pushServer *httptest.Server
			var u *url.URL

			t.Cleanup(func() {
				db.Close()
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

			db, err = webpush.ConnectToDatabase()

			if err != nil {
				t.Fatalf("TestHandlePush err = %v, wantErr = %v", err, nil)
			}

			db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

			if err != nil {
				t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
			}

			// save test subscription
			if _, err = db.NewRaw("INSERT INTO subscription (client_id, recipient_id, endpoint) VALUES (?, ?, ?)", rec.Subscription.ClientId, rec.Subscription.RecipientId, rec.Subscription.Endpoint).Exec(ctx); err != nil {
				t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
			}

			// save test keys
			if _, err = db.NewRaw("INSERT INTO keys (p256dh, auth_secret, subscription_endpoint) VALUES (?, ?, ?)", rec.Subscription.Keys.P256DH, rec.Subscription.Keys.Auth, rec.Subscription.Endpoint).Exec(ctx); err != nil {
				t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
			}

			applicationServer = httptest.NewServer(http.HandlerFunc(HandlePush))

			if u, err = url.Parse(applicationServer.URL); err != nil {
				t.Fatalf("HandlePush err = %v, wantErr = %v", err, nil)
			}

			query := u.Query()

			query.Add("id", tt.params.RecipientId)

			if tt.params.WithWebPushParams == nil {
				tt.params.WithWebPushParams = &webpush.WithWebPushParams{}
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

			res, err := applicationServer.Client().Do(req)

			if err != nil {
				t.Fatalf("HandlePush err = %v, wantErr = %v", err, nil)
			}

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}
