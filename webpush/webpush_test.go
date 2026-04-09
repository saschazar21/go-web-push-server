package webpush

import (
	"crypto/ecdh"
	"encoding/base64"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saschazar21/go-web-push-server/models"
	"github.com/saschazar21/go-web-push-server/request"
	webpush_test "github.com/saschazar21/go-web-push-server/test"
	"github.com/saschazar21/go-web-push-server/utils"
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
	t.Setenv(utils.SKIP_PADDING_ENV, "true") // needed, because RFC8291 values are unpadded

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
		saltBuf         [SALT_SIZE]byte
		saltDec         []byte
		sharedSecretBuf []byte
	)

	plainTextClientKey, err := base64.RawURLEncoding.DecodeString(clientKey)
	if err != nil {
		t.Fatalf("failed to decode client public key: %v", err)
	}

	if clientPubKey, err = decodePublicKey(plainTextClientKey); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	if privKey, err = testDecodePrivateKey(privateKey); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	plainTextPublicKey, err := base64.RawURLEncoding.DecodeString(publicKey)
	if err != nil {
		t.Fatalf("failed to decode public key: %v", err)
	}
	pubKeyCmp, _ := decodePublicKey(plainTextPublicKey)

	assert.Equal(t, pubKeyCmp, privKey.PublicKey())

	if authSecretBuf, err = base64.RawURLEncoding.DecodeString(authSecret); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	if saltDec, err = base64.RawURLEncoding.DecodeString(salt); err != nil {
		t.Errorf(errMsg, err, nil)
	}

	copy(saltBuf[:], saltDec)

	p := &webpushDetails{
		authSecret: authSecretBuf,
		clientKey:  clientPubKey,
		privateKey: privKey,
		salt:       saltBuf,
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
		CEK:       cekBuf,
		Endpoint:  "",
		Nonce:     nonceBuf,
		PublicKey: privKey.PublicKey(),
		Salt:      saltBuf,
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
	t.Setenv(utils.VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(utils.VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
`)
	t.Setenv(utils.VAPID_SUBJECT_ENV, "test@example.com")

	testServer := httptest.NewServer(http.HandlerFunc(webpush_test.EchoHeaders))

	t.Cleanup(func() {
		testServer.Close()
	})

	var (
		authSecret = "BTBZMqHH6r4Tts7J_aSIgg"
		clientKey  = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"

		emptyBuffer     = make([]byte, 0)
		timestamp       = time.Now().Add(time.Hour)
		invalidEndpoint = "htp://push.example"
	)

	decodedP256DH, err := base64.RawURLEncoding.DecodeString(clientKey)
	if err != nil {
		t.Fatalf("failed to decode p256dh key: %v", err)
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(authSecret)
	if err != nil {
		t.Fatalf("failed to decode auth secret: %v", err)
	}

	type test struct {
		name         string
		subscription *models.PushSubscription
		wantErr      bool
		wantReqErr   bool
	}

	tests := []test{
		{
			"validates",
			&models.PushSubscription{
				Endpoint:       (*utils.EncryptedString)(&testServer.URL),
				ExpirationTime: (*utils.EpochMillis)(&timestamp),
				Keys: &models.SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			false,
			false,
		},
		{
			"fails at malformatted endpoint",
			&models.PushSubscription{
				Endpoint:       (*utils.EncryptedString)(&invalidEndpoint),
				ExpirationTime: (*utils.EpochMillis)(&timestamp),
				Keys: &models.SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&decodedP256DH),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
				},
			},
			false,
			true,
		},
		{
			"fails at malformatted public key",
			&models.PushSubscription{
				Endpoint:       (*utils.EncryptedString)(&testServer.URL),
				ExpirationTime: (*utils.EpochMillis)(&timestamp),
				Keys: &models.SubscriptionKeys{
					P256DH:     (*utils.EncryptedBytes)(&emptyBuffer),
					AuthSecret: (*utils.EncryptedBytes)(&decodedAuthSecret),
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

				if res, err = p.Send([]byte("hello, world"), &request.WithWebPushParams{TTL: 300}); (err != nil) != tt.wantReqErr {
					t.Errorf("TestWebPush err = %v, wantErr = %v", err, tt.wantErr)
				}

				if err == nil {
					log.Println(res.Header)

					assert.NotNil(t, res.Header.Get("Authorization"))
					assert.Equal(t, "aes128gcm", res.Header.Get("Content-Encoding"))
				}

			}
		})
	}
}
