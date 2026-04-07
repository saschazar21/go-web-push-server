package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/saschazar21/go-web-push-server/errors"
	"github.com/saschazar21/go-web-push-server/models"
	"github.com/saschazar21/go-web-push-server/request"
	"github.com/saschazar21/go-web-push-server/utils"
	"github.com/saschazar21/go-web-push-server/vapid"
	"golang.org/x/crypto/hkdf"
)

const (
	CEK_SIZE    = 16
	IKM_SIZE    = 32
	NONCE_SIZE  = 12
	RECORD_SIZE = 4
	SALT_SIZE   = 16

	MAX_PAYLOAD_SIZE   = 4096
	MAX_PLAINTEXT_SIZE = 3993 // see https://datatracker.ietf.org/doc/html/rfc8291/#section-4
)

type WebPush struct {
	CEK       []byte
	Endpoint  string
	Nonce     []byte
	PublicKey *ecdh.PublicKey
	Salt      [SALT_SIZE]byte
}

func (p *WebPush) encrypt(payload []byte) (buf []byte, err error) {
	buf = payload

	if len(buf) > MAX_PLAINTEXT_SIZE {
		errorPayload := errors.NewErrorResponse(http.StatusRequestEntityTooLarge, "Push message body is too large", fmt.Sprintf("For compatibility reasons, the push message body must not exceed %d bytes.", MAX_PLAINTEXT_SIZE))

		return nil, errors.NewResponseError(errorPayload, http.StatusRequestEntityTooLarge)
	}

	padding := make([]byte, MAX_PLAINTEXT_SIZE-len(buf))

	if os.Getenv(utils.SKIP_PADDING_ENV) != "" {
		padding = make([]byte, 0)
	}

	buf = append(buf, padding...)
	buf = append(buf, 0x02)

	var block cipher.Block

	if block, err = aes.NewCipher(p.CEK); err != nil {
		return nil, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var gcm cipher.AEAD

	if gcm, err = cipher.NewGCM(block); err != nil {
		return nil, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	enc := gcm.Seal(buf[:0], p.Nonce, buf, nil)

	return enc, err
}

// https://datatracker.ietf.org/doc/html/rfc8188#section-2.1
func (p *WebPush) generateEncryptionContentCodingHeader() (header []byte) {
	rs := [RECORD_SIZE]byte{0x00, 0x00, 0x10, 0x00} // 4096 bytes, see https://datatracker.ietf.org/doc/html/rfc8291#section-4
	keyid := p.PublicKey.Bytes()                    // 65 bytes uncompressed public key format (0x04 || X || Y)
	idlen := byte(len(keyid))                       // 65

	header = append(p.Salt[:], rs[:]...)
	header = append(header, idlen)
	header = append(header, keyid...)

	return
}

func (p *WebPush) Encrypt(payload []byte) (buf []byte, err error) {
	buf = p.generateEncryptionContentCodingHeader()

	var cipher []byte

	if cipher, err = p.encrypt(payload); err != nil {
		return nil, err
	}

	return append(buf, cipher...), nil
}

func (p *WebPush) Send(payload []byte, params *request.WithWebPushParams) (res *http.Response, err error) {
	var buf []byte

	if buf, err = p.Encrypt(payload); err != nil {
		return
	}

	req := request.WebPushRequest{
		Endpoint:          p.Endpoint,
		Payload:           buf,
		WithWebPushParams: params,
		WithSalt: &request.WithSalt{
			Salt: p.Salt[:],
		},
		WithPublicKey: &request.WithPublicKey{
			PublicKey: p.PublicKey,
		},
	}

	return req.Send()
}

type webpushDetails struct {
	authSecret []byte
	clientKey  *ecdh.PublicKey
	privateKey *ecdh.PrivateKey
	salt       [16]byte
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

		return ikm, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func (p *webpushDetails) generatePseudoRandomKey() (prk []byte, err error) {
	var ikm []byte

	if ikm, err = p.generateInputKeyingMaterial(); err != nil {
		return
	}

	prk = hkdf.Extract(sha256.New, ikm, p.salt[:])

	return
}

func (p *webpushDetails) generateSharedSecret() (buf []byte, err error) {
	if buf, err = p.privateKey.ECDH(p.clientKey); err != nil {
		log.Println(err)

		return buf, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
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

		return nil, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
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

		return nil, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func generateSalt() (salt [16]byte, err error) {
	salt = [SALT_SIZE]byte{}

	if _, err = rand.Read(salt[:]); err != nil {
		log.Println(err)

		return salt, fmt.Errorf("failed to generate salt")
	}

	return
}

func generatePrivateKey() (key *ecdh.PrivateKey, err error) {
	k, err := vapid.GenerateVapidKey()
	if err != nil {
		log.Println(err)

		return key, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	if key, err = k.PrivateKey.ECDH(); err != nil {
		log.Println(err)

		return key, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	return
}

func NewWebPush(sub *models.PushSubscription) (p *WebPush, err error) {
	if sub.Endpoint == nil {
		log.Printf("No endpoint provided for subscription %s\n", sub)
		return p, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusBadRequest)
	}

	if sub.Keys == nil || sub.Keys.AuthSecret == nil || sub.Keys.P256DH == nil {
		log.Printf("No keys provided for subscription %s\n", sub)
		return p, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusBadRequest)
	}

	var clientKey *ecdh.PublicKey

	if clientKey, err = decodePublicKey(*sub.Keys.P256DH); err != nil {
		log.Println(err)

		return p, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	var privateKey *ecdh.PrivateKey

	if privateKey, err = generatePrivateKey(); err != nil {
		return
	}

	var salt [16]byte

	if salt, err = generateSalt(); err != nil {
		log.Println(err)

		return p, errors.NewResponseError(errors.INTERNAL_SERVER_ERROR, http.StatusInternalServerError)
	}

	details := webpushDetails{
		authSecret: *sub.Keys.AuthSecret,
		clientKey:  clientKey,
		privateKey: privateKey,
		salt:       salt,
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

	return &WebPush{
		CEK:       cek,
		Endpoint:  string(*sub.Endpoint),
		Nonce:     nonce,
		PublicKey: privateKey.PublicKey(),
		Salt:      salt,
	}, err
}
