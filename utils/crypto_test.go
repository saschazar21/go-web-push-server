package utils

import "testing"

func TestEncrypt(t *testing.T) {
	t.Setenv(MASTER_KEY_ENV, "T5p2WRcCKFSA6vhXlBEqyDBxNsWHSkydLadEhLL1eGc=")

	original := "Hello, World!"
	encrypted, err := Encrypt([]byte(original))
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != original {
		t.Errorf("Decrypted value does not match original. Got: %s, want: %s", decrypted, original)
	}
}
