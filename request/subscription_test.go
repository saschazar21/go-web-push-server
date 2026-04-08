package request

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
)

func TestParseSubscriptionReqeuest(t *testing.T) {

	recipientSubscription := &utils.RecipientSubscription{}

	if err := webpush_test.LoadFixture("fcm.json", recipientSubscription); err != nil {
		t.Fatalf("failed to load subscription fixture: %v", err)
	}

	type testCase struct {
		name      string
		recipient *utils.Recipient
		wantErr   bool
	}

	tests := []testCase{
		{
			name: "should parse valid subscription request",
			recipient: &utils.Recipient{
				ClientId:     "test client",
				RecipientId:  "test user",
				Subscription: recipientSubscription,
			},
			wantErr: false,
		},
		{
			name: "should return error on missing client ID",
			recipient: &utils.Recipient{
				ClientId:     "",
				RecipientId:  "test user",
				Subscription: recipientSubscription,
			},
			wantErr: true,
		},
		{
			name: "should return error on missing recipient ID",
			recipient: &utils.Recipient{
				ClientId:     "test client",
				RecipientId:  "",
				Subscription: recipientSubscription,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.recipient)

			if err != nil {
				t.Fatalf("failed to marshal recipient: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "https:///api/v1/subscribe", bufio.NewReader(bytes.NewReader(payload)))
			req.Header.Set("content-type", utils.APPLICATION_JSON)

			sub, err := ParseSubscriptionRequest(req)

			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error but got nil")
			}

			if err == nil {
				if sub.ClientId != tc.recipient.ClientId {
					t.Errorf("expected client ID %s but got %s", tc.recipient.ClientId, sub.ClientId)
				}

				if sub.RecipientId != tc.recipient.RecipientId {
					t.Errorf("expected recipient ID %s but got %s", tc.recipient.RecipientId, sub.RecipientId)
				}
			}
		})
	}
}
