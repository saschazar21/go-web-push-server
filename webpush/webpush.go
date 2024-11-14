package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/hkdf"
)

const (
	CEK_SIZE   = 16
	IKM_SIZE   = 32
	NONCE_SIZE = 12
	SALT_SIZE  = 16

	MAX_PAYLOAD_SIZE   = 4096
	MAX_PLAINTEXT_SIZE = 3993 // see https://datatracker.ietf.org/doc/html/rfc8291/#section-4
)

type webpush struct {
	CEK       []byte
	Endpoint  string
	Nonce     []byte
	PublicKey *ecdh.PublicKey
	Salt      []byte
}

func (p *webpush) encrypt(plaintext fmt.Stringer) (buf []byte, err error) {
	buf = []byte(plaintext.String())

	if len(buf) > MAX_PLAINTEXT_SIZE {
		errorPayload := NewErrorResponse(http.StatusRequestEntityTooLarge, "Push message body is too large", fmt.Sprintf("For compatibility reasons, the push message body must not exceed %d bytes.", MAX_PLAINTEXT_SIZE))

		return nil, NewResponseError(errorPayload, http.StatusRequestEntityTooLarge)
	}

	padding := make([]byte, MAX_PLAINTEXT_SIZE-len(buf))

	if os.Getenv(SKIP_PADDING_ENV) != "" {
		padding = make([]byte, 0)
	}

	buf = append(buf, 0x02)
	buf = append(buf, padding...)

	var block cipher.Block

	if block, err = aes.NewCipher(p.CEK); err != nil {
		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var gcm cipher.AEAD

	if gcm, err = cipher.NewGCM(block); err != nil {
		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	enc := gcm.Seal(buf[:0], p.Nonce, buf, nil)

	return enc, err
}

// https://datatracker.ietf.org/doc/html/rfc8188#section-2.1
func (p *webpush) generateEncryptionContentCodingHeader() (header []byte) {
	rs := []byte{0x00, 0x00, 0x10, 0x00} // 4096
	keyid := p.PublicKey.Bytes()
	idlen := byte(len(keyid)) // 56

	header = append(p.Salt, rs...)
	header = append(header, idlen)
	header = append(header, keyid...)

	return
}

func (p *webpush) Encrypt(plaintext fmt.Stringer) (buf []byte, err error) {
	buf = p.generateEncryptionContentCodingHeader()

	var cipher []byte

	if cipher, err = p.encrypt(plaintext); err != nil {
		return nil, err
	}

	return append(buf, cipher...), nil
}

type webpushDetails struct {
	authSecret []byte
	clientKey  *ecdh.PublicKey
	privateKey *ecdh.PrivateKey
	salt       []byte
}

func (p *webpushDetails) generateInputKeyingMaterial() (ikm []byte, err error) {
	clientKey := p.clientKey.Bytes()
	pubKey := p.privateKey.PublicKey().Bytes()

	keyInfo := append([]byte("WebPush: info"), 0x00)
	keyInfo = append(keyInfo, clientKey...)
	keyInfo = append(keyInfo, pubKey...)

	var sharedSecret []byte

	if sharedSecret, err = p.generateSharedSecret(); err != nil {
		return
	}

	if ikm, err = deriveKey(sharedSecret, p.authSecret, keyInfo, IKM_SIZE); err != nil {
		log.Println(err)

		return ikm, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func (p *webpushDetails) generatePseudoRandomKey() (prk []byte, err error) {
	var ikm []byte

	if ikm, err = p.generateInputKeyingMaterial(); err != nil {
		return
	}

	prk = hkdf.Extract(sha256.New, ikm, p.salt)

	return
}

func (p *webpushDetails) generateSharedSecret() (buf []byte, err error) {
	if buf, err = p.privateKey.ECDH(p.clientKey); err != nil {
		log.Println(err)

		return buf, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func generateContentEncryptionKey(prk []byte) (cek []byte, err error) {
	cek = make([]byte, CEK_SIZE)

	cekInfo := append([]byte("Content-Encoding: aes128gcm"), 0x00)

	reader := hkdf.Expand(sha256.New, prk, cekInfo)

	if _, err = reader.Read(cek); err != nil {
		log.Println("generating CEK failed:")
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func generateNonce(prk []byte) (nonce []byte, err error) {
	nonce = make([]byte, NONCE_SIZE)

	nonceInfo := append([]byte("Content-Encoding: nonce"), 0x00)

	reader := hkdf.Expand(sha256.New, prk, nonceInfo)

	if _, err = reader.Read(nonce); err != nil {
		log.Println("generating Nonce failed:")
		log.Println(err)

		return nil, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func generateSalt() (salt []byte, err error) {
	salt = make([]byte, SALT_SIZE)

	if _, err = rand.Read(salt); err != nil {
		log.Println(err)

		return salt, fmt.Errorf("failed to generate salt")
	}

	return
}

func getPrivateKey() (key *ecdh.PrivateKey, err error) {
	pemEncoded := os.Getenv(VAPID_PRIVATE_KEY_ENV)

	var k *vapidKey

	if k, err = DecodeFromPEM(pemEncoded); err != nil {
		log.Println(err)

		return key, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	if key, err = k.PrivateKey.ECDH(); err != nil {
		log.Println(err)

		return key, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func NewWebPush(sub *pushSubscription) (p *webpush, err error) {
	var authSecret []byte

	if authSecret, err = base64.RawURLEncoding.DecodeString(sub.Keys.Auth); err != nil {
		log.Println(err)

		return p, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var clientKey *ecdh.PublicKey

	if clientKey, err = decodePublicKey(sub.Keys.P256DH); err != nil {
		log.Println(err)

		return p, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var privateKey *ecdh.PrivateKey

	if privateKey, err = getPrivateKey(); err != nil {
		return
	}

	var salt []byte

	if salt, err = generateSalt(); err != nil {
		log.Println(err)

		return p, NewResponseError(INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	details := webpushDetails{
		authSecret,
		clientKey,
		privateKey,
		salt,
	}

	var cek, nonce, prk []byte

	if prk, err = details.generatePseudoRandomKey(); err != nil {
		return
	}

	if cek, err = generateContentEncryptionKey(prk); err != nil {
		return
	}

	if nonce, err = generateNonce(prk); err != nil {
		return
	}

	return &webpush{
		cek,
		sub.Endpoint,
		nonce,
		privateKey.PublicKey(),
		salt,
	}, err
}
