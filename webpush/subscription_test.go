package webpush

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscription(t *testing.T) {
	keys := pushSubscriptionKeys{
		P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := pushSubscription{
		Endpoint:       "https://example.com",
		ExpirationTime: 0,
		Keys:           keys,
	}

	sub := recipient{
		ClientId:      "test client",
		Subject:       "test user",
		Subscriptions: []pushSubscription{pushSub},
	}

	type test struct {
		name    string
		payload recipient
		cmp     *recipient
		wantErr bool
	}

	tests := []test{
		{
			"validates recipient",
			sub,
			&sub,
			false,
		},
		{
			"fails to validate missing client_id",
			recipient{
				Subject:       "test user",
				Subscriptions: []pushSubscription{pushSub},
			},
			nil,
			true,
		},
		{
			"fails to validate missing subject",
			recipient{
				ClientId:      "test client",
				Subscriptions: []pushSubscription{pushSub},
			},
			nil,
			true,
		},
		{
			"fails to validate missing endpoint",
			recipient{
				ClientId: "test client",
				Subject:  "test user",
				Subscriptions: []pushSubscription{
					{Keys: keys},
				},
			},
			nil,
			true,
		},
		{
			"fails to validate missing p256dh key",
			recipient{
				ClientId: "test client",
				Subject:  "test user",
				Subscriptions: []pushSubscription{
					{Keys: pushSubscriptionKeys{
						Auth: keys.Auth,
					},
					},
				},
			},
			nil,
			true,
		},
		{
			"fails to validate missing auth key",
			recipient{
				ClientId: "test client",
				Subject:  "test user",
				Subscriptions: []pushSubscription{
					{Keys: pushSubscriptionKeys{
						P256DH: keys.P256DH,
					},
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
				t.Errorf("TestSubscription err = %v, wantEerr = %v", err, nil)
			}

			req := httptest.NewRequest(http.MethodPost, "http://example.com", bytes.NewBuffer(enc))
			req.Header = http.Header{
				http.CanonicalHeaderKey("content-type"): {APPLICATION_JSON},
			}

			sub, err := ParseSubscription(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestSubscription err = %v, wantErr = %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.cmp, sub)
		})
	}
}
