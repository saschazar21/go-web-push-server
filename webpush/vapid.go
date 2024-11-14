package webpush

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DEFAULT_VAPID_EXPIRY_DURATION int64 = 86400

func getExpiryEpoch() (epoch *Epoch) {
	env := os.Getenv(VAPID_EXPIRY_DURATION_ENV)

	exp, err := strconv.ParseInt(env, 10, 64)

	if err != nil {
		log.Printf("failed to parse %s env, falling back to default: %d\n", VAPID_EXPIRY_DURATION_ENV, DEFAULT_VAPID_EXPIRY_DURATION)
		exp = DEFAULT_VAPID_EXPIRY_DURATION
	}

	if exp <= 0 {
		log.Printf("%s env must be a positive integer > 0, falling back to default: %d\n", VAPID_EXPIRY_DURATION_ENV, DEFAULT_VAPID_EXPIRY_DURATION)
		exp = DEFAULT_VAPID_EXPIRY_DURATION
	}

	return &Epoch{
		time.Now().UTC().Add(time.Duration(exp) * time.Second),
	}
}

func NewVAPID(aud string) (signedToken string, pubKey string, err error) {
	sub := fmt.Sprintf("mailto:%s", os.Getenv(VAPID_SUBJECT_ENV))
	raw := os.Getenv(VAPID_PRIVATE_KEY_ENV)
	key := new(vapidKey)

	if key, err = DecodeFromPEM(raw); err != nil {
		return
	}

	claims := &vapidClaims{
		sub,
		aud,
		getExpiryEpoch(),
	}

	if err = claims.Validate(); err != nil {
		log.Println(err)

		return signedToken, pubKey, fmt.Errorf("failed to validate VAPID JWT token")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	if signedToken, err = token.SignedString(key.PrivateKey); err != nil {
		log.Println(err)

		return signedToken, pubKey, fmt.Errorf("failed to sign VAPID JWT token")
	}

	pubKey = key.String()

	return
}
