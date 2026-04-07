package v1

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/saschazar21/go-web-push-server/auth"
	"github.com/saschazar21/go-web-push-server/db"
	"github.com/saschazar21/go-web-push-server/models"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun/extra/bundebug"
)

func TestHandleUnsubscribe(t *testing.T) {
	basicAuthPassword := "123"
	t.Setenv(auth.BASIC_AUTH_PASSWORD_ENV, basicAuthPassword)
	t.Setenv("CWD", "../../")

	type params struct {
		clientId    string
		recipientId string
	}

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
		name       string
		method     string
		payload    params
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
			"should return 405 Method Not Allowed on invalid method",
			http.MethodGet,
			params{
				clientId:    "test client",
				recipientId: "test user",
			},
			405,
		},
		{
			"should return 401 Unauthorized on invalid client",
			http.MethodDelete,
			params{
				clientId:    "",
				recipientId: "test user",
			},
			401,
		},
		{
			"should return 404 Not Found on missing client",
			http.MethodDelete,
			params{
				clientId:    "missing client",
				recipientId: "test user",
			},
			404,
		},
		{
			"should return 204 No Content on successful recipient unsubscribe",
			http.MethodDelete,
			params{
				clientId:    "test client",
				recipientId: "test user",
			},
			204,
		},
		{
			"should return 204 No Content on successful client unsubscribe",
			http.MethodDelete,
			params{
				clientId:    "test client",
				recipientId: "",
			},
			204,
		},
	}

	// create test recipient
	rec := recipient{
		Subscription: subscription{
			ClientId:    "test client",
			RecipientId: "test user",
			Endpoint:    "https://example.com",
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

	// save test subscription
	sub := &models.PushSubscription{
		ClientId:    rec.Subscription.ClientId,
		RecipientId: rec.Subscription.RecipientId,
		Endpoint:    (*utils.EncryptedString)(&rec.Subscription.Endpoint),
		Keys: &models.SubscriptionKeys{
			P256DH:     (*utils.EncryptedBytes)(&decodedClientKey),
			AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
		},
	}

	conn, err := db.Connect()

	conn.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	if _, err := conn.NewInsert().Model(sub).Exec(ctx); err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	if sub.Keys != nil {
		sub.Keys.PushSubscriptionHash = sub.Hash
		if _, err := conn.NewInsert().Model(sub.Keys).Exec(ctx); err != nil {
			t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
		}
	}

	conn.Close()

	if err = c.Snapshot(ctx, postgres.WithSnapshotName("unsubscribe")); err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			var err error
			var u *url.URL

			t.Cleanup(func() {
				server.Close()
				c.Restore(ctx, postgres.WithSnapshotName("unsubscribe"))
			})

			server = httptest.NewServer(http.HandlerFunc(HandleUnsubscribe))

			if u, err = url.Parse(server.URL); err != nil {
				t.Fatalf("HandleUnsubscribe err = %v, wantErr = %v", err, nil)
			}

			query := u.Query()

			query.Add("client", tt.payload.clientId)
			query.Add("id", tt.payload.recipientId)

			u.RawQuery = query.Encode()

			log.Printf("URL: %s", u.String())

			// create request
			req, err := http.NewRequest(tt.method, u.String(), nil)

			if err != nil {
				t.Fatalf("HandleUnsubscribe err = %v, wantErr = %v", err, nil)
			}

			req.SetBasicAuth(tt.payload.clientId, basicAuthPassword)

			// create response recorder
			res, err := server.Client().Do(req)

			if err != nil {
				t.Fatalf("HandleUnsubscribe err = %v, wantErr = %v", err, nil)
			}

			// check status code
			if res.StatusCode != tt.wantStatus {
				defer res.Body.Close()

				buffer, _ := io.ReadAll(res.Body)

				t.Errorf("HandleUnsubscribe body = %v", string(buffer))
				t.Errorf("HandleUnsubscribe status = %v, wantStatus = %v", res.StatusCode, tt.wantStatus)
			}
		})
	}
}
