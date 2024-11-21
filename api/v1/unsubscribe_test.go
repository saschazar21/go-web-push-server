package v1

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/webpush"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun/extra/bundebug"
)

func TestHandleUnsubscribe(t *testing.T) {
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
			"should return 400 Bad Request on invalid client",
			http.MethodDelete,
			params{
				clientId:    "",
				recipientId: "test user",
			},
			400,
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

	db, err := webpush.ConnectToDatabase()

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

	db.Close()

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
