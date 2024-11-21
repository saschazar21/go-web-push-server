package webpush

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

func TestSubscription(t *testing.T) {
	keys := recipientKeys{
		P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := recipientSubscription{
		Endpoint:       "https://example.com",
		ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
		Keys:           &keys,
	}

	sub := recipient{
		ClientId:     "test client",
		RecipientId:  "test user",
		Subscription: &pushSub,
	}

	type test struct {
		name    string
		payload any
		cmp     *pushSubscription
		wantErr bool
	}

	tests := []test{
		{
			"validates recipient",
			sub,
			&pushSubscription{
				ClientId:       "test client",
				RecipientId:    "test user",
				Endpoint:       "https://example.com",
				ExpirationTime: pushSub.ExpirationTime,
				Keys:           &pushSubscriptionKeys{P256DH: keys.P256DH, Auth: keys.Auth},
			},
			false,
		},
		{
			"validates missing recipient id",
			recipient{
				ClientId:     "test client",
				Subscription: &pushSub,
			},
			&pushSubscription{
				ClientId:       "test client",
				Endpoint:       "https://example.com",
				ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
				Keys:           &pushSubscriptionKeys{P256DH: keys.P256DH, Auth: keys.Auth},
			},
			false,
		},
		{
			"fails to validate missing client_id",
			recipient{
				RecipientId:  "test user",
				Subscription: &pushSub,
			},
			nil,
			true,
		},
		{
			"fails to validate missing endpoint",
			recipient{
				ClientId:    "test client",
				RecipientId: "test user",
				Subscription: &recipientSubscription{
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys:           &keys,
				},
			},
			nil,
			true,
		},
		{
			"fails to validate missing p256dh key",
			recipient{
				ClientId:    "test client",
				RecipientId: "test user",
				Subscription: &recipientSubscription{
					Endpoint:       "https://example.com",
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys: &recipientKeys{
						Auth: keys.Auth,
					},
				},
			},
			nil,
			true,
		},
		{
			"fails to validate missing auth key",
			recipient{
				ClientId:    "test client",
				RecipientId: "test user",
				Subscription: &recipientSubscription{
					Endpoint:       "https://example.com",
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys: &recipientKeys{
						P256DH: keys.P256DH,
					},
				},
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := json.Marshal(tt.payload)

			if err != nil {
				t.Errorf("TestSubscription err = %v, wantErr = %v", err, nil)
			}

			req := httptest.NewRequest(http.MethodPost, "http://example.com", bytes.NewBuffer(enc))
			req.Header = http.Header{
				http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
			}

			sub, err := ParseSubscription(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestSubscription err = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.cmp != nil {
				assert.Equal(t, tt.cmp.ClientId, sub.ClientId)
				assert.Equal(t, tt.cmp.RecipientId, sub.RecipientId)
				assert.Equal(t, tt.cmp.Endpoint, sub.Endpoint)
				assert.Equal(t, tt.cmp.Keys.P256DH, sub.Keys.P256DH)
			}
		})
	}
}

func TestSubscriptionWithDB(t *testing.T) {
	type test struct {
		name    string
		exec    func(ctx context.Context, conn *bun.DB, sub *pushSubscription) error
		wantErr bool
	}

	ctx := context.Background()

	keys := pushSubscriptionKeys{
		P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := pushSubscription{
		Endpoint:       "https://example.com",
		ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
		Keys:           &keys,

		ClientId:    "test client",
		RecipientId: "test user",
	}

	c, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	tests := []test{
		{
			"should fetch recipient from database",
			func(ctx context.Context, conn *bun.DB, sub *pushSubscription) (err error) {
				if err = sub.Save(ctx, conn); err != nil {
					return
				}

				var result []pushSubscription

				result, err = GetSubscriptionsByClientAndRecipient(ctx, conn, sub.ClientId, sub.RecipientId)

				if err != nil {
					return
				}

				r := result[0]

				assert.Equal(t, pushSub.ClientId, r.ClientId)
				assert.Equal(t, pushSub.RecipientId, r.RecipientId)
				assert.Equal(t, pushSub.Endpoint, r.Endpoint)
				assert.Equal(t, pushSub.Keys.P256DH, r.Keys.P256DH)

				if err = r.Delete(ctx, conn); err != nil {
					return
				}

				return
			},
			false,
		},
		{
			"should delete recipient from database",
			func(ctx context.Context, conn *bun.DB, sub *pushSubscription) (err error) {
				if err = sub.Save(ctx, conn); err != nil {
					return
				}

				var result []pushSubscription

				result, err = GetSubscriptionsByClient(ctx, conn, sub.ClientId)

				if err != nil {
					return
				}

				r := result[0]

				assert.Equal(t, pushSub.ClientId, r.ClientId)
				assert.Equal(t, pushSub.RecipientId, r.RecipientId)
				assert.Equal(t, pushSub.Endpoint, r.Endpoint)
				assert.Equal(t, pushSub.Keys.P256DH, r.Keys.P256DH)

				if err = DeleteSubscriptionsByClient(ctx, conn, sub.ClientId); err != nil {
					return
				}

				return
			},
			false,
		},
		{
			"should delete recipient by client id and recipient id from database",
			func(ctx context.Context, conn *bun.DB, sub *pushSubscription) (err error) {
				if err = sub.Save(ctx, conn); err != nil {
					return
				}

				var result []pushSubscription

				result, err = GetSubscriptionsByClient(ctx, conn, sub.ClientId)

				if err != nil {
					return
				}

				r := result[0]

				assert.Equal(t, pushSub.ClientId, r.ClientId)
				assert.Equal(t, pushSub.RecipientId, r.RecipientId)
				assert.Equal(t, pushSub.Endpoint, r.Endpoint)
				assert.Equal(t, pushSub.Keys.P256DH, r.Keys.P256DH)

				if err = DeleteSubscriptionsByClientAndRecipient(ctx, conn, sub.ClientId, sub.RecipientId); err != nil {
					return
				}

				return
			},
			false,
		},
		{
			"should save recipient without id to database",
			func(ctx context.Context, conn *bun.DB, sub *pushSubscription) (err error) {
				cp := *sub
				cp.RecipientId = ""

				if err = cp.Save(ctx, conn); err != nil {
					return
				}

				var result []pushSubscription

				result, err = GetSubscriptionsByClient(ctx, conn, sub.ClientId)

				if err != nil {
					return
				}

				r := result[0]

				assert.Equal(t, pushSub.ClientId, r.ClientId)
				assert.Equal(t, pushSub.Endpoint, r.Endpoint)
				assert.Equal(t, pushSub.Keys.P256DH, r.Keys.P256DH)
				assert.NotEmpty(t, r.RecipientId)

				if err = DeleteSubscriptionsByClient(ctx, conn, sub.ClientId); err != nil {
					return
				}

				return
			},
			false,
		},
		{
			"fails to store recipient with missing client id in database",
			func(ctx context.Context, conn *bun.DB, sub *pushSubscription) (err error) {
				cp := *sub
				cp.ClientId = ""

				if err = cp.Save(ctx, conn); err != nil {
					return
				}

				return
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var conn *bun.DB

			t.Cleanup(func() {
				if err := conn.Close(); err != nil {
					t.Fatalf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
				}

				if err := c.Restore(ctx); err != nil {
					t.Fatalf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
				}
			})

			if conn, err = ConnectToDatabase(); err != nil {
				t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
			}

			conn.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

			err := tt.exec(ctx, conn, &pushSub)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
