package models

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/saschazar21/go-web-push-server/db"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
)

const (
	TEST_CLIENT_ID    = "test-client-id"
	TEST_RECIPIENT_ID = "test-recipient-id"
)

var (
	ONE_HOUR_AGO   = time.Now().Add(-time.Hour)
	ONE_HOUR_LATER = time.Now().Add(time.Hour)
)

func TestSubscription(t *testing.T) {
	t.Setenv(utils.HMAC_SECRET_KEY_ENV, "T5p2WRcCKFSA6vhXlBEqyDBxNsWHSkydLadEhLL1eGc=")
	t.Setenv(utils.MASTER_KEY_ENV, "l342tf9eC2l4/fVytEkkzQzYyqd3eKd6GViw65WB5yI=")

	ctx := context.Background()

	container, err := webpush_test.CreateContainer(ctx, t)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	defer container.Terminate(ctx)

	fixture := &utils.RecipientSubscription{}
	if err := webpush_test.LoadFixture("mozilla.json", fixture); err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	fixture.ExpirationTime = (*utils.EpochMillis)(&ONE_HOUR_LATER)

	decodedP256DH, err := base64.RawURLEncoding.DecodeString(fixture.Keys.P256DH)
	if err != nil {
		t.Fatalf("failed to decode p256dh key: %v", err)
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(fixture.Keys.Auth)
	if err != nil {
		t.Fatalf("failed to decode auth secret: %v", err)
	}

	type testCase struct {
		name         string
		subscription *PushSubscription
		wantErr      bool
	}

	testCases := []testCase{
		{
			name: "valid subscription",
			subscription: &PushSubscription{
				ClientId:       TEST_CLIENT_ID,
				RecipientId:    TEST_RECIPIENT_ID,
				Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
				ExpirationTime: fixture.ExpirationTime,
				Keys: &SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid subscription with missing client ID",
			subscription: &PushSubscription{
				RecipientId:    TEST_RECIPIENT_ID,
				Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
				ExpirationTime: fixture.ExpirationTime,
				Keys: &SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid subscription with missing recipient ID",
			subscription: &PushSubscription{
				ClientId:       TEST_CLIENT_ID,
				Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
				ExpirationTime: fixture.ExpirationTime,
				Keys: &SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid subscription with missing endpoint",
			subscription: &PushSubscription{
				ClientId:       TEST_CLIENT_ID,
				RecipientId:    TEST_RECIPIENT_ID,
				ExpirationTime: fixture.ExpirationTime,
				Keys: &SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid subscription with missing keys",
			subscription: &PushSubscription{
				ClientId:       TEST_CLIENT_ID,
				RecipientId:    TEST_RECIPIENT_ID,
				Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
				ExpirationTime: fixture.ExpirationTime,
			},
			wantErr: true,
		},
		{
			name: "invalid subscription with invalid expiration time",
			subscription: &PushSubscription{
				ClientId:       TEST_CLIENT_ID,
				RecipientId:    TEST_RECIPIENT_ID,
				Endpoint:       (*utils.EncryptedString)(&fixture.Endpoint),
				ExpirationTime: (*utils.EpochMillis)(&ONE_HOUR_AGO),
				Keys: &SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			wantErr: true,
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

			if err := tc.subscription.Save(ctx, conn); (err != nil) != tc.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				pushSubscriptions := make([]*PushSubscription, 0)
				hash := utils.Hash([]byte(*tc.subscription.Hash))
				enc := base64.StdEncoding.EncodeToString(hash[:])
				if _, err := GetSubscriptionByHash(ctx, conn, enc); err != nil {
					t.Errorf("GetSubscriptionByHash() error = %v, wantErr %v", err, false)
				}

				if pushSubscriptions, err = GetSubscriptionsByClientId(ctx, conn, TEST_CLIENT_ID); err != nil {
					t.Errorf("GetSubscriptionByClientId() error = %v, wantErr %v", err, false)
				}

				if len(pushSubscriptions) != 1 {
					t.Errorf("expected 1 subscription, got %d", len(pushSubscriptions))
				}

				if pushSubscriptions, err = GetSubscriptionsByClientIdAndRecipientId(ctx, conn, TEST_CLIENT_ID, TEST_RECIPIENT_ID); err != nil {
					t.Errorf("GetSubscriptionsByClientIdAndRecipientId() error = %v, wantErr %v", err, false)
				}

				if len(pushSubscriptions) != 1 {
					t.Errorf("expected 1 subscription, got %d", len(pushSubscriptions))
				}

				if err := DeleteSubscriptionByEndpoint(ctx, conn, string(*tc.subscription.Endpoint)); err != nil {
					t.Errorf("DeleteSubscriptionByEndpoint() error = %v, wantErr %v", err, false)
				}

				count, _ := conn.NewSelect().
					Model((*PushSubscription)(nil)).
					Count(ctx)

				if count != 0 {
					t.Errorf("expected 0 subscriptions, got %d", count)
				}

				if err := tc.subscription.Save(ctx, conn); err != nil {
					t.Errorf("Save() error = %v, wantErr %v", err, false)
				}

				if err := DeleteSubscriptionsByClientId(ctx, conn, TEST_CLIENT_ID); err != nil {
					t.Errorf("DeleteSubscriptionsByClientId() error = %v, wantErr %v", err, false)
				}

				count, _ = conn.NewSelect().
					Model((*PushSubscription)(nil)).
					Count(ctx)

				if count != 0 {
					t.Errorf("expected 0 subscriptions, got %d", count)
				}

				if err := tc.subscription.Save(ctx, conn); err != nil {
					t.Errorf("Save() error = %v, wantErr %v", err, false)
				}

				if err := DeleteSubscriptionsByClientIdAndRecipientId(ctx, conn, TEST_CLIENT_ID, TEST_RECIPIENT_ID); err != nil {
					t.Errorf("DeleteSubscriptionsByClientIdAndRecipientId() error = %v, wantErr %v", err, false)
				}

				count, _ = conn.NewSelect().
					Model((*PushSubscription)(nil)).
					Count(ctx)

				if count != 0 {
					t.Errorf("expected 0 subscriptions, got %d", count)
				}
			}
		})
	}
}
