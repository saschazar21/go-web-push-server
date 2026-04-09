package utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"
)

type EncryptedBytes []byte

func (e EncryptedBytes) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}

	encrypted, err := Encrypt(e)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt bytes: %w", err)
	}
	return encrypted, nil
}

func (e *EncryptedBytes) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		decrypted, err := Decrypt(v)
		if err != nil {
			return fmt.Errorf("failed to decrypt bytes: %w", err)
		}
		*e = decrypted
	case string:
		decrypted, err := Decrypt([]byte(v))
		if err != nil {
			return fmt.Errorf("failed to decrypt bytes: %w", err)
		}
		*e = decrypted
	default:
		return fmt.Errorf("unsupported type for EncryptedBytes: %T", v)
	}

	return nil
}

type EncryptedString string

func (e EncryptedString) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}

	encrypted, err := Encrypt([]byte(e))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt string: %w", err)
	}
	return encrypted, nil
}

func (e *EncryptedString) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		decrypted, err := Decrypt(v)
		if err != nil {
			return fmt.Errorf("failed to decrypt string: %w", err)
		}
		*e = EncryptedString(decrypted)
	case string:
		decrypted, err := Decrypt([]byte(v))
		if err != nil {
			return fmt.Errorf("failed to decrypt string: %w", err)
		}
		*e = EncryptedString(decrypted)
	default:
		return fmt.Errorf("unsupported type for EncryptedString: %T", v)
	}

	return nil
}

type Epoch time.Time

func (e Epoch) MarshalJSON() ([]byte, error) {
	ts := time.Time(e).Unix()
	return []byte(fmt.Sprintf("%d", ts)), nil
}

func (e *Epoch) UnmarshalJSON(data []byte) (err error) {
	var epoch int64
	stringified := string(data)

	if epoch, err = strconv.ParseInt(stringified, 10, 64); err != nil {
		return fmt.Errorf("failed to parse epoch: %w", err)
	}

	*e = Epoch(time.Unix(epoch, 0))

	return
}

var _ sql.Scanner = (*Epoch)(nil)

func (e *Epoch) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*e = Epoch(v)
	default:
		return fmt.Errorf("unsupported type for Epoch: %T", v)
	}

	return nil
}

var _ driver.Valuer = (*Epoch)(nil)

func (e Epoch) Value() (driver.Value, error) {
	return time.Time(e), nil
}

type EpochMillis time.Time

func (e EpochMillis) MarshalJSON() ([]byte, error) {
	ms := time.Time(e).UnixMilli()
	return []byte(fmt.Sprintf("%d", ms)), nil
}

func (e *EpochMillis) UnmarshalJSON(data []byte) (err error) {
	var epoch int64
	stringified := string(data)

	if epoch, err = strconv.ParseInt(stringified, 10, 64); err != nil {
		return fmt.Errorf("failed to parse epoch: %w", err)
	}

	*e = EpochMillis(time.UnixMilli(epoch))

	return
}

var _ sql.Scanner = (*EpochMillis)(nil)

func (e *EpochMillis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*e = EpochMillis(v)
	default:
		return fmt.Errorf("unsupported type for EpochMillis: %T", v)
	}

	return nil
}

var _ driver.Valuer = (*EpochMillis)(nil)

func (e EpochMillis) Value() (driver.Value, error) {
	return time.Time(e), nil
}

type HashedString string

var _ driver.Valuer = (*HashedString)(nil)

func (h HashedString) Value() (driver.Value, error) {
	if len(h) == 0 {
		return nil, nil
	}

	hashed := Hash([]byte(h))

	return hashed[:], nil
}

var _ sql.Scanner = (*HashedString)(nil)

func (h *HashedString) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		s := base64.RawURLEncoding.EncodeToString(v)
		*h = HashedString(s)
	case string:
		*h = HashedString(v)
	default:
		return fmt.Errorf("unsupported type for HashedString: %T", v)
	}

	return nil
}

func (h *HashedString) Compare(plain []byte) bool {
	if len(*h) == 0 {
		return false
	}

	hashed := Hash(plain)
	str := base64.RawURLEncoding.EncodeToString(hashed[:])

	return string(*h) == str
}

func (h HashedString) String() string {
	if len(h) == 0 {
		return ""
	}
	return string(h)
}

type RecipientKeys struct {
	P256DH string `json:"p256dh" validate:"len=87"`
	Auth   string `json:"auth" validate:"len=22"`
}

type RecipientSubscription struct {
	Endpoint       string         `json:"endpoint" validate:"http_url"`
	ExpirationTime *EpochMillis   `json:"expirationTime,omitempty"`
	Keys           *RecipientKeys `json:"keys" validate:"required"`
}

type Recipient struct {
	ClientId    string `json:"clientId" validate:"required"`
	RecipientId string `json:"id" validate:"required"`

	Subscription *RecipientSubscription `json:"subscription" validate:"required"`
}

type StringerValidator interface {
	String() string
	Validate() error
}
