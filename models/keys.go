package models

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/uptrace/bun"
)

type SubscriptionKeys struct {
	bun.BaseModel `bun:"table:webpush_keys,alias:pk"`

	Hash           *utils.HashedString   `json:"hash" bun:"p256dh_hash,type:bytea,pk"`
	P256DH         *utils.EncryptedBytes `json:"-" validate:"required" bun:"p256dh,type:bytea,notnull"`
	AuthSecretHash *utils.HashedString   `json:"-" bun:"auth_secret_hash,type:bytea,notnull,unique"`
	AuthSecret     *utils.EncryptedBytes `json:"-" validate:"required" bun:"auth_secret,type:bytea,notnull"`

	PushSubscriptionHash *utils.HashedString `json:"-" validate:"required" bun:"subscription_hash,type:bytea,notnull,unique"`
}

var _ bun.BeforeAppendModelHook = (*SubscriptionKeys)(nil)

func (k *SubscriptionKeys) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	if k.P256DH != nil {
		p256dhHash := utils.HashedString(*k.P256DH)
		k.Hash = &p256dhHash
	}

	if k.AuthSecret != nil {
		authSecretHash := utils.HashedString(*k.AuthSecret)
		k.AuthSecretHash = &authSecretHash
	}

	return nil
}

func (k *SubscriptionKeys) Save(ctx context.Context, db bun.IDB) (err error) {
	if err = k.Validate(); err != nil {
		return
	}

	errMsg := "Failed to store subscription keys in database"

	run := func(ctx context.Context, db bun.Tx) error {
		_, _ = db.NewDelete().
			Model((*SubscriptionKeys)(nil)).
			Where("subscription_hash = ?", k.PushSubscriptionHash).
			Exec(ctx)

		rows, err := db.NewInsert().
			Model(k).
			On("CONFLICT (p256dh_hash) DO UPDATE").
			Set("auth_secret = EXCLUDED.auth_secret").
			Set("auth_secret_hash = EXCLUDED.auth_secret_hash").
			Set("subscription_hash = EXCLUDED.subscription_hash").
			Exec(ctx)
		if err != nil {
			log.Printf("inserting subscription keys failed: %v", err)
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, err.Error())
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		affected, err := rows.RowsAffected()
		if err != nil {
			log.Printf("checking affected rows after inserting subscription key failed: %v", err)
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, err.Error())
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		if affected == 0 {
			log.Printf("inserting subscription key failed: no rows affected")
			payload := errors.NewErrorResponse(http.StatusInternalServerError, errMsg, "no rows affected")
			return errors.NewResponseError(payload, http.StatusInternalServerError)
		}

		return nil
	}

	if tx, ok := db.(bun.Tx); ok {
		err = run(ctx, tx)
	} else {
		err = db.RunInTx(ctx, nil, run)
	}

	return
}

func (k SubscriptionKeys) String() string {
	return fmt.Sprintf("[Subscription Key] %s", k.Hash)
}

func (k SubscriptionKeys) Validate() (err error) {
	if err = utils.CustomValidateStruct(k); err != nil {
		log.Printf("invalid Subscription Key: %v", err)
		payload := errors.NewErrorResponse(http.StatusBadRequest, "Invalid Subscription Key", err.Error())
		return errors.NewResponseError(payload, http.StatusBadRequest)
	}

	return
}
