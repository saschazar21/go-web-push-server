package webpush

import (
	"crypto/ecdh"
	"encoding/base64"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testDecodePrivateKey(enc string) (privKey *ecdh.PrivateKey, err error) {
	var buf []byte

	if buf, err = base64.RawURLEncoding.DecodeString(enc); err != nil {
		return
	}

	curve := ecdh.P256()

	if privKey, err = curve.NewPrivateKey(buf); err != nil {
		return
	}

	return
}

// fixtures taken from Appendix A. of https://datatracker.ietf.org/doc/rfc8291/
func TestWebPushFixtures(t *testing.T) {
	t.Setenv(SKIP_PADDING_ENV, "true") // needed, because RFC8291 values are unpadded

	var errMsg = "TestWebPush err = %v, wantErr = %v"

	var (
		plainText = "When I grow up, I want to be a watermelon"
		cipher    = "8pfeW0KbunFT06SuDKoJH9Ql87S1QUrdirN6GcG7sFz1y1sqLgVi1VhjVkHsUoEsbI_0LpXMuGvnzQ"
		result    = "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	)

	var (
		nonce        = "4h_95klXJ5E_qnoN"
		cek          = "oIhVW04MRdy2XN9CiKLxTg"
		ikm          = "S4lYMb_L0FxCeq0WhDx813KgSYqU26kOyzWUdsXYyrg"
		prk          = "09_eUZGrsvxChDCGRCdkLiDXrReGOEVeSCdCcPBSJSc"
		sharedSecret = "kyrL1jIIOHEzg3sM2ZWRHDRB62YACZhhSlknJ672kSs"
	)

	var (
		authSecret = "BTBZMqHH6r4Tts7J_aSIgg"
		clientKey  = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"

		publicKey  = "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8"
		privateKey = "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"

		salt = "DGv6ra1nlYgDCS1FRnbzlw"
	)

	var (
		authSecretBuf   []byte
		cekBuf          []byte
		cipherBuf       []byte
		clientPubKey    *ecdh.PublicKey
		err             error
		ikmBuf          []byte
		nonceBuf        []byte
		privKey         *ecdh.PrivateKey
		prkBuf          []byte
		resultBuf       []byte
		saltBuf         []byte
		sharedSecretBuf []byte
	)

	if clientPubKey, err = decodePublicKey(clientKey); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	if privKey, err = testDecodePrivateKey(privateKey); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	pubKeyCmp, _ := decodePublicKey(publicKey)

	assert.Equal(t, pubKeyCmp, privKey.PublicKey())

	if authSecretBuf, err = base64.RawURLEncoding.DecodeString(authSecret); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	if saltBuf, err = base64.RawURLEncoding.DecodeString(salt); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	p := &webpushDetails{
		authSecretBuf,
		clientPubKey,
		privKey,
		saltBuf,
	}

	if sharedSecretBuf, err = p.generateSharedSecret(); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	sharedSecretEnc := base64.RawURLEncoding.EncodeToString(sharedSecretBuf)

	assert.Equal(t, sharedSecret, sharedSecretEnc)

	if ikmBuf, err = p.generateInputKeyingMaterial(); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	ikmEnc := base64.RawURLEncoding.EncodeToString(ikmBuf)

	assert.Equal(t, ikm, ikmEnc)

	if prkBuf, err = p.generatePseudoRandomKey(); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	prkEnc := base64.RawURLEncoding.EncodeToString(prkBuf)

	assert.Equal(t, prk, prkEnc)

	if cekBuf, err = generateContentEncryptionKey(prkBuf); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	cekEnc := base64.RawURLEncoding.EncodeToString(cekBuf)

	assert.Equal(t, cek, cekEnc)

	if nonceBuf, err = generateNonce(prkBuf); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	nonceEnc := base64.RawURLEncoding.EncodeToString(nonceBuf)

	assert.Equal(t, nonce, nonceEnc)

	push := &WebPush{
		cekBuf,
		"",
		nonceBuf,
		privKey.PublicKey(),
		saltBuf,
	}

	headerBuf := push.generateEncryptionContentCodingHeader()

	assert.Len(t, headerBuf, 86)

	if cipherBuf, err = push.encrypt([]byte(plainText)); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	cipherEnc := base64.RawURLEncoding.EncodeToString(cipherBuf)

	assert.Equal(t, cipher, cipherEnc)

	if resultBuf, err = push.Encrypt([]byte(plainText)); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	resultEnc := base64.RawURLEncoding.EncodeToString(resultBuf)

	assert.Equal(t, result, resultEnc)
}

func TestWebPush(t *testing.T) {
	t.Setenv(VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
`)
	t.Setenv(VAPID_SUBJECT_ENV, "test@example.com")

	testServer := httptest.NewServer(http.HandlerFunc(handleRequest))

	defer func() {
		testServer.Close()
	}()

	type test struct {
		name         string
		subscription *PushSubscription
		wantErr      bool
		wantReqErr   bool
	}

	tests := []test{
		{
			"validates",
			&PushSubscription{
				Endpoint:       testServer.URL,
				ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
				Keys: &pushSubscriptionKeys{
					P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
					Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
				},
			},
			false,
			false,
		},
		{
			"fails at malformatted endpoint",
			&PushSubscription{
				Endpoint:       "htp://push.example",
				ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
				Keys: &pushSubscriptionKeys{
					P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
					Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
				},
			},
			false,
			true,
		},
		{
			"fails at malformatted public key",
			&PushSubscription{
				Endpoint:       testServer.URL,
				ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
				Keys: &pushSubscriptionKeys{
					P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TC",
					Auth:   "DGv6ra1nlYgDCS1FRnbzlw",
				},
			},
			true,
			false,
		},
		{
			"fails at malformatted auth secret",
			&PushSubscription{
				Endpoint:       testServer.URL,
				ExpirationTime: &EpochMillis{time.Time(time.Now().Add(time.Hour))},
				Keys: &pushSubscriptionKeys{
					P256DH: "BPZ_GnkGFYfUcY0D0yMWcAQIuvQfV5tSw_dd7iIQktNR1dhdDflA1eQyJT-0ZSwpDO43mNbBwogEMTh7TCSkuP0",
					Auth:   "DGv6ra1nlYgDCS1FRnbzlw153",
				},
			},
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res *http.Response
			var err error

			var p *WebPush

			if p, err = NewWebPush(tt.subscription); (err != nil) != tt.wantErr {
				t.Errorf("TestWebPush err = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {

				if res, err = p.Send([]byte("hello, world"), &WithWebPushParams{TTL: 300}); (err != nil) != tt.wantReqErr {
					t.Errorf("TestWebPush err = %v, wantErr = %v", err, tt.wantErr)
				}

				if err == nil {
					log.Println(res.Header)

					assert.NotNil(t, res.Header.Get("Authorization"))
					assert.Equal(t, "aesgcm", res.Header.Get("Content-Encoding"))
					assert.Regexp(t, `^salt=[A-Za-z0-9-_]+$`, res.Header.Get("Encryption"))
				}

			}
		})
	}
}
