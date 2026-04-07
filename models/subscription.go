package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/uptrace/bun"
)

type PushSubscription struct {
	bun.BaseModel `bun:"table:webpush_subscriptions,alias:ps"`

	Hash           *utils.HashedString    `json:"hash" bun:"endpoint_hash,type:bytea,pk"`
	Endpoint       *utils.EncryptedString `json:"-" validate:"http_url" bun:"endpoint,type:bytea,notnull"`
	ClientId       string                 `json:"clientId" validate:"required" bun:"client_id,notnull"`
	RecipientId    string                 `json:"recipientId" validate:"required" bun:"recipient_id,notnull"`
	ExpirationTime *utils.EpochMillis     `json:"expirationTime,omitempty" validate:"omitempty,epoch-gt-now" bun:"expiration_time"`

	Keys *SubscriptionKeys `validate:"-" bun:"rel:has-one,join:endpoint_hash=subscription_hash"`
}

var _ bun.BeforeAppendModelHook = (*PushSubscription)(nil)

func (s *PushSubscription) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	if s.Endpoint != nil {
		endpointHash := utils.HashedString(*s.Endpoint)
		s.Hash = &endpointHash
	}
	return nil
}

func (s *PushSubscription) Save(ctx context.Context, db bun.IDB) (err error) {
	keys := &SubscriptionKeys{}

	if s.Keys == nil {
		log.Printf("%s has no keys, aborting save", s)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "Subscription keys are required")
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	keys = s.Keys
	s.Keys = nil

	if err = s.Validate(); err != nil {
		return
	}

	errMsg := "Failed to store subscription in database"

	run := func(ctx context.Context, db bun.Tx) error {
		rows, err := db.NewInsert().
			Model(s).
			On("CONFLICT (endpoint_hash) DO UPDATE").
			Set("endpoint = EXCLUDED.endpoint").
			Set("client_id = EXCLUDED.client_id").
			Set("recipient_id = EXCLUDED.recipient_id").
			Set("expiration_time = EXCLUDED.expiration_time").
			Exec(ctx)
		if err != nil {
			log.Printf("inserting subscription failed: %v", err)
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, err.Error())
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		affected, err := rows.RowsAffected()
		if err != nil {
			log.Printf("checking affected rows after inserting subscription failed: %v", err)
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, err.Error())
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		if affected == 0 {
			log.Printf("inserting subscription failed: no rows affected")
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, "no rows affected")
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		if keys != nil {
			keys.PushSubscriptionHash = s.Hash
			if err = keys.Save(ctx, db); err != nil {
				return err
			}
		}

		s.Keys = keys
		return nil
	}

	if tx, ok := db.(bun.Tx); ok {
		err = run(ctx, tx)
	} else {
		err = db.RunInTx(ctx, nil, run)
	}

	return
}

func (s PushSubscription) String() string {
	return fmt.Sprintf("[Push Subscription] %s (Client: %s, Recipient: %s)", s.Hash, s.ClientId, s.RecipientId)
}

func (s PushSubscription) Validate() (err error) {
	if err = utils.CustomValidateStruct(s); err != nil {
		log.Printf("invalid subscription: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "Invalid subscription contents", err.Error())
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	return
}

func DeleteSubscriptionByEndpoint(ctx context.Context, db bun.IDB, endpoint string) (err error) {
	if _, err = db.NewDelete().
		Model((*PushSubscription)(nil)).
		Where("endpoint_hash = ?", utils.HashedString(endpoint)).
		Exec(ctx); err != nil {
		log.Printf("deleting subscription by endpoint failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to delete subscription", err.Error())
		return errors.NewResponseError(payload, http.StatusInternalServerError)
	}
	return nil
}

func DeleteSubscriptionByHash(ctx context.Context, db bun.IDB, hash string) (err error) {
	if _, err = db.NewDelete().
		Model((*PushSubscription)(nil)).
		Where("endpoint_hash = ?", hash).
		Exec(ctx); err != nil {
		log.Printf("deleting subscription by hash failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to delete subscription", err.Error())
		return errors.NewResponseError(payload, http.StatusInternalServerError)
	}
	return nil
}

func DeleteSubscriptionsByClientId(ctx context.Context, db bun.IDB, clientId string) (err error) {
	if _, err = db.NewDelete().
		Model((*PushSubscription)(nil)).
		Where("client_id = ?", clientId).
		Exec(ctx); err != nil {
		log.Printf("deleting subscriptions by client ID failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to delete subscriptions", err.Error())
		return errors.NewResponseError(payload, http.StatusInternalServerError)
	}
	return nil
}

func DeleteSubscriptionsByClientIdAndRecipientId(ctx context.Context, db bun.IDB, clientId, recipientId string) (err error) {
	if _, err = db.NewDelete().
		Model((*PushSubscription)(nil)).
		Where("client_id = ?", clientId).
		Where("recipient_id = ?", recipientId).
		Exec(ctx); err != nil {
		log.Printf("deleting subscriptions by client ID and recipient ID failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to delete subscriptions", err.Error())
		return errors.NewResponseError(payload, http.StatusInternalServerError)
	}
	return nil
}

func GetSubscriptionByHash(ctx context.Context, db bun.IDB, hash string) (subscription *PushSubscription, err error) {
	subscription = &PushSubscription{}

	decoded, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		log.Printf("decoding subscription hash failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "Invalid subscription hash", err.Error())
		return nil, errors.NewResponseError(payload, http.StatusBadRequest)
	}

	if err = db.NewSelect().
		Model(subscription).
		Where("endpoint_hash = ?", decoded).
		Where("expiration_time IS NULL OR expiration_time > ?", time.Now().UTC()).
		Relation("Keys").
		Scan(ctx); err != nil {
		log.Printf("fetching subscription by hash failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch subscription", err.Error())
		return nil, errors.NewResponseError(payload, http.StatusInternalServerError)
	}

	return subscription, nil
}

func GetSubscriptionsByClientId(ctx context.Context, db bun.IDB, clientId string) (subscriptions []*PushSubscription, err error) {
	subscriptions = make([]*PushSubscription, 0)

	if err = db.NewSelect().
		Model(&subscriptions).
		Where("client_id = ?", clientId).
		Where("expiration_time IS NULL OR expiration_time > ?", time.Now().UTC()).
		Relation("Keys").
		Scan(ctx); err != nil {
		log.Printf("fetching subscriptions by client ID failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch subscriptions", err.Error())
		return nil, errors.NewResponseError(payload, http.StatusInternalServerError)
	}

	return subscriptions, nil
}

func GetSubscriptionsByClientIdAndRecipientId(ctx context.Context, db bun.IDB, clientId, recipientId string) (subscriptions []*PushSubscription, err error) {
	subscriptions = make([]*PushSubscription, 0)

	if err = db.NewSelect().
		Model(&subscriptions).
		Where("client_id = ?", clientId).
		Where("recipient_id = ?", recipientId).
		Where("expiration_time IS NULL OR expiration_time > ?", time.Now().UTC()).
		Relation("Keys").
		Scan(ctx); err != nil {
		log.Printf("fetching subscriptions by client ID and recipient ID failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to fetch subscriptions", err.Error())
		return nil, errors.NewResponseError(payload, http.StatusInternalServerError)
	}

	return subscriptions, nil
}

func HasExistingSubscriptionsByClientId(ctx context.Context, db bun.IDB, clientId string) (exists bool, err error) {
	exists, err = db.NewSelect().
		Model((*PushSubscription)(nil)).
		Where("client_id = ?", clientId).
		Where("expiration_time IS NULL OR expiration_time > ?", time.Now().UTC()).
		Exists(ctx)
	if err != nil {
		log.Printf("checking for existing subscriptions by client ID failed: %v", err)
		payload := errors.NewErrorResponse(http.StatusInternalServerError, "Failed to check existing subscriptions", err.Error())
		return false, errors.NewResponseError(payload, http.StatusInternalServerError)
	}

	return
}
