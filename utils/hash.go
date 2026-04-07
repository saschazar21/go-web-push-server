package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"os"
)

func Hash(data []byte) (hashed [32]byte) {
	secret := os.Getenv(HMAC_SECRET_KEY_ENV)

	decoded, err := base64.StdEncoding.DecodeString(secret)
	if len(decoded) == 0 || err != nil {
		if err != nil {
			log.Printf("failed to decode HMAC_SECRET_KEY env, make sure it's a valid base64-encoding: %v", err)
		}
		if secret == "" {
			log.Printf("HMAC_SECRET_KEY env not set, falling back to plain SHA-256...")
		}

		hashed = sha256.Sum256(data)
		return
	}

	// If the secret is set and valid, we can use it to create a more secure hash
	mac := hmac.New(sha256.New, decoded)
	mac.Write(data)
	copy(hashed[:], mac.Sum(nil))

	return
}
