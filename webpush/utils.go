package webpush

import (
	"crypto/ecdh"
	"crypto/sha256"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/hkdf"
)

func decodePublicKey(buf []byte) (pubKey *ecdh.PublicKey, err error) {
	curve := ecdh.P256()

	if pubKey, err = curve.NewPublicKey(buf); err != nil {
		log.Println(err)
		return pubKey, fmt.Errorf("failed to decode bytes into ECDH public key")
	}

	return
}

func deriveKey(secret, salt, info []byte, length uint) (buf []byte, err error) {
	var written int

	reader := hkdf.New(sha256.New, secret, salt, info)

	buf = make([]byte, length)

	if written, err = io.ReadFull(reader, buf); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to read derived key into buffer")
	}

	if uint(written) != length {
		return nil, fmt.Errorf("expected buffer length does not match actual buffer length")
	}

	return
}
