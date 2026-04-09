package models

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/saschazar21/go-web-push-server/db"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
)

func TestKeys(t *testing.T) {
	t.Setenv(utils.HMAC_SECRET_KEY_ENV, "T5p2WRcCKFSA6vhXlBEqyDBxNsWHSkydLadEhLL1eGc=")
	t.Setenv(utils.MASTER_KEY_ENV, "l342tf9eC2l4/fVytEkkzQzYyqd3eKd6GViw65WB5yI=")

	ctx := context.Background()

	container, err := webpush_test.CreateContainer(ctx, t)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	defer container.Terminate(ctx)

	fixture := &utils.RecipientSubscription{}
	if err := webpush_test.LoadFixture("fcm.json", fixture); err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	conn, err := db.Connect()
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	subscription := &PushSubscription{
		ClientId:       TEST_CLIENT_ID,
		RecipientId:    TEST_RECIPIENT_ID,
		Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
		ExpirationTime: fixture.ExpirationTime,
	}

	if _, err := conn.NewInsert().
		Model(subscription).
		Exec(ctx); err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	conn.Close()

	decodedP256DH, err := base64.RawURLEncoding.DecodeString(fixture.Keys.P256DH)
	if err != nil {
		t.Fatalf("failed to decode p256dh key: %v", err)
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(fixture.Keys.Auth)
	if err != nil {
		t.Fatalf("failed to decode auth secret: %v", err)
	}

	type testCase struct {
		name    string
		keys    *SubscriptionKeys
		wantErr bool
	}

	testCases := []testCase{
		{
			name: "valid keys",
			keys: &SubscriptionKeys{
				P256DH:               (*utils.EncryptedBytes)(&decodedP256DH),
				AuthSecret:           (*utils.EncryptedBytes)(&decodedAuthSecret),
				PushSubscriptionHash: subscription.Hash,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := db.Connect()
			if err != nil {
				t.Fatalf("failed to connect to database: %v", err)
			}

			t.Cleanup(func() {
				conn.Close()
				container.Restore(ctx)
			})

			if err := tc.keys.Save(ctx, conn); (err != nil) != tc.wantErr {
				t.Errorf("SubscriptionKeys.Save() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				keys := make([]*SubscriptionKeys, 0)
				err := conn.NewSelect().
					Model((*SubscriptionKeys)(nil)).
					Scan(ctx, &keys)
				if err != nil {
					t.Errorf("failed to retrieve keys: %v", err)
				}

				if len(keys) != 1 {
					t.Errorf("expected 1 keys record, got %d", len(keys))
				}
			}
		})
	}
}
