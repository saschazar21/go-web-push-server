package webpush

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
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
	keys := pushSubscriptionKeys{
		P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := pushSubscription{
		ClientId:       "test client",
		RecipientId:    "test user",
		Endpoint:       "https://example.com",
		ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
		Keys:           keys,
	}

	sub := recipient{
		ClientId:     "test client",
		RecipientId:  "test user",
		Subscription: &pushSub,
	}

	type test struct {
		name    string
		payload recipient
		cmp     *pushSubscription
		wantErr bool
	}

	tests := []test{
		{
			"validates recipient",
			sub,
			&pushSub,
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
				Keys:           keys,
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
				Subscription: &pushSubscription{
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys:           keys,
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
				Subscription: &pushSubscription{
					Endpoint:       "https://example.com",
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys: pushSubscriptionKeys{
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
				Subscription: &pushSubscription{
					Endpoint:       "https://example.com",
					ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
					Keys: pushSubscriptionKeys{
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
	ctx := context.Background()

	keys := pushSubscriptionKeys{
		P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := pushSubscription{
		Endpoint:       "https://example.com",
		ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
		Keys:           keys,

		ClientId:    "test client",
		RecipientId: "test user",
	}

	c, err := webpush_test.CreateContainer(ctx, t)

	if err != nil {
		t.Fatalf("TestPostgres err = %v, wantErr = %v", err, nil)
	}

	var conn *bun.DB

	if conn, err = ConnectToDatabase(); err != nil {
		t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
	}

	conn.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	t.Cleanup(func() {
		c.Terminate(ctx)
	})

	t.Run("should save recipient to database", func(t *testing.T) {
		t.Cleanup(func() {
			if err := c.Restore(ctx); err != nil {
				t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
			}
		})

		if err = pushSub.Save(ctx, conn); err != nil {
			t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", err, nil)
		}

		result, resultErr := GetSubscriptionsByClientAndRecipient(ctx, conn, pushSub.ClientId, pushSub.RecipientId)

		if resultErr != nil {
			t.Errorf("TestSubscriptionWithDB err = %v, wantErr = %v", resultErr, nil)
		}

		r := result[0]

		log.Println(r)

		assert.Equal(t, pushSub.ClientId, r.ClientId)
		assert.Equal(t, pushSub.RecipientId, r.RecipientId)
		assert.Equal(t, pushSub.Endpoint, r.Endpoint)
		assert.Equal(t, pushSub.Keys.P256DH, r.Keys.P256DH)
	})
}
