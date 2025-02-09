package webpush

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

type pushSubscriptionKeys struct {
	bun.BaseModel `bun:"table:keys,alias:k"`

	P256DH string `json:"p256dh" validate:"len=87" bun:"p256dh,pk"`
	Auth   string `json:"auth" validate:"len=22" bun:"auth_secret,notnull"`

	Endpoint string `json:"-" bun:"subscription_endpoint,notnull"`
}

func (k *pushSubscriptionKeys) Save(ctx context.Context, db bun.IDB) (err error) {
	if _, err = db.
		NewInsert().
		Model(k).
		On("CONFLICT (p256dh) DO UPDATE").
		Set("auth_secret = EXCLUDED.auth_secret").
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func (k pushSubscriptionKeys) String() string {
	return fmt.Sprintf("pushSubscriptionKeys{p256dh: %s, auth: %s}", k.P256DH, k.Auth)
}

type PushSubscription struct {
	bun.BaseModel `bun:"table:subscription,alias:sub"`

	Endpoint       string       `json:"endpoint" validate:"http_url" bun:"endpoint,pk"`
	ExpirationTime *EpochMillis `json:"expirationTime,omitempty" validate:"omitempty,epoch-gt-now" bun:"expiration_time,nullzero"`
	ClientId       string       `json:"clientId" validate:"required" bun:"client_id,notnull"`
	RecipientId    string       `json:"recipientId" bun:"recipient_id,notnull"`

	Keys *pushSubscriptionKeys `json:"keys" validate:"required" bun:"rel:has-one,join:endpoint=subscription_endpoint"`
}

func (s *PushSubscription) Delete(ctx context.Context, db bun.IDB) (err error) {
	if _, err = db.
		NewDelete().
		Model(s).
		WherePK().
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func (s *PushSubscription) Save(ctx context.Context, db bun.IDB) (err error) {
	if err = s.Validate(); err != nil {
		return
	}

	if s.RecipientId == "" {
		s.RecipientId = fmt.Sprintf("anonymous_%s", xid.New().String())
	}

	s.Keys.Endpoint = s.Endpoint

	if _, err = db.
		NewInsert().
		Model(s).
		On("CONFLICT (endpoint) DO UPDATE").
		Set("expiration_time = EXCLUDED.expiration_time").
		Set("client_id = EXCLUDED.client_id").
		Set("recipient_id = EXCLUDED.recipient_id").
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	if err = s.Keys.Save(ctx, db); err != nil {
		return
	}

	return
}

func (s PushSubscription) String() string {
	return fmt.Sprintf("PushSubscription{endpoint: %s, expirationTime: %s, clientId: %s, recipientId: %s, keys: %s}",
		s.Endpoint, s.ExpirationTime, s.ClientId, s.RecipientId, s.Keys)
}

func (s *PushSubscription) Validate() (err error) {
	if err = CustomValidateStruct(s); err != nil {
		log.Println(err)

		payload := NewErrorResponse(http.StatusBadRequest, "invalid subscription contents", err.Error())
		return NewResponseError(payload, http.StatusBadRequest)
	}

	return
}

func DeleteSubscriptionsByClient(ctx context.Context, db bun.IDB, clientId string) (err error) {
	if _, err = db.
		NewDelete().
		Model(&PushSubscription{}).
		Where("client_id = ?", clientId).
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func DeleteSubscriptionsByClientAndRecipient(ctx context.Context, db bun.IDB, clientId, recipientId string) (err error) {
	if _, err = db.
		NewDelete().
		Model(&PushSubscription{}).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.
				Where("client_id = ?", clientId).
				Where("recipient_id = ?", recipientId)
		}).
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func DeleteSubscriptionByEndpoint(ctx context.Context, db bun.IDB, endpoint string) (err error) {
	if _, err = db.
		NewDelete().
		Model(&PushSubscription{}).
		Where("endpoint = ?", endpoint).
		Exec(ctx); err != nil {
		log.Println(err)

		return NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func GetSubscriptionsByClient(ctx context.Context, db bun.IDB, clientId string) (subs []PushSubscription, err error) {
	if err = db.
		NewSelect().
		Model(&subs).
		Relation("Keys").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("client_id = ?", clientId).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.
						Where("expiration_time IS NULL").
						WhereOr("expiration_time > NOW()")
				})
		}).
		Scan(ctx); err != nil {
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func GetSubscriptionsByClientAndRecipient(ctx context.Context, db bun.IDB, clientId, recipientId string) (subs []PushSubscription, err error) {
	if err = db.
		NewSelect().
		Model(&subs).
		Relation("Keys").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("client_id = ?", clientId).
				Where("recipient_id = ?", recipientId).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.
						Where("expiration_time IS NULL").
						WhereOr("expiration_time > NOW()")
				})
		}).
		Scan(ctx); err != nil {
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func HasExistingSubscriptionsByClient(ctx context.Context, db bun.IDB, clientId string) (exists bool, err error) {
	if exists, err = db.
		NewSelect().
		Model((*PushSubscription)(nil)).
		Where("client_id = ?", clientId).
		Exists(ctx); err != nil {
		log.Println(err)

		return false, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func ParseSubscription(req *http.Request) (sub *PushSubscription, err error) {
	r := new(recipient)

	if err = ParseBody(req, r); err != nil {
		log.Println(err)

		responseErr, ok := err.(ResponseError)

		if !ok {
			payload := NewErrorResponse(http.StatusBadRequest, "failed to decode recipient")
			return sub, NewResponseError(payload, http.StatusBadRequest)
		}

		return sub, responseErr
	}

	if err = r.Validate(); err != nil {
		return
	}

	sub = &PushSubscription{
		Endpoint:       r.Subscription.Endpoint,
		ExpirationTime: r.Subscription.ExpirationTime,
		ClientId:       r.ClientId,
		RecipientId:    r.RecipientId,
		Keys: &pushSubscriptionKeys{
			P256DH: r.Subscription.Keys.P256DH,
			Auth:   r.Subscription.Keys.Auth,
		},
	}

	if err = sub.Validate(); err != nil {
		return nil, err
	}

	return
}
