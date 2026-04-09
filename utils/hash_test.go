package utils

import "testing"

func TestHash(t *testing.T) {
	t.Setenv(HMAC_SECRET_KEY_ENV, "T5p2WRcCKFSA6vhXlBEqyDBxNsWHSkydLadEhLL1eGc=")

	data := []byte("Hello, World!")
	hashed1 := Hash(data)
	hashed2 := Hash(data)

	if hashed1 != hashed2 {
		t.Errorf("Hash function is not deterministic. Got: %x and %x", hashed1, hashed2)
	}
}

func TestHashWithoutSecret(t *testing.T) {
	t.Setenv(HMAC_SECRET_KEY_ENV, "")

	data := []byte("Hello, World!")
	hashed1 := Hash(data)
	hashed2 := Hash(data)

	if hashed1 != hashed2 {
		t.Errorf("Hash function is not deterministic without secret. Got: %x and %x", hashed1, hashed2)
	}
}
