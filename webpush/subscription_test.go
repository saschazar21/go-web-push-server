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
		"BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
		"DGv6ra1nlYgDCS1FRnbzlw",
	}

	pushSub := pushSubscription{
		"https://example.com",
		0,
		keys,
	}

	sub := subscription{
		"test client",
		"test user",
		pushSub,
	}

	type test struct {
		name    string
		payload subscription
		cmp     *subscription
		wantErr bool
	}

	tests := []test{
		{
			"validates subscription",
			sub,
			&sub,
			false,
		},
		{
			"fails to validate missing client_id",
			subscription{
				Subject:      "test user",
				Subscription: pushSub,
			},
			nil,
			true,
		},
		{
			"fails to validate missing subject",
			subscription{
				ClientId:     "test client",
				Subscription: pushSub,
			},
			nil,
			true,
		},
		{
			"fails to validate missing endpoint",
			subscription{
				ClientId: "test client",
				Subject:  "test user",
				Subscription: pushSubscription{
					Keys: keys,
				},
			},
			nil,
			true,
		},
		{
			"fails to validate missing p256dh key",
			subscription{
				ClientId: "test client",
				Subject:  "test user",
				Subscription: pushSubscription{
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
			subscription{
				ClientId: "test client",
				Subject:  "test user",
				Subscription: pushSubscription{
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
