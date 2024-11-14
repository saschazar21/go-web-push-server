package webpush

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
)

const CURVE_BIT_SIZE = 256

type vapidKey struct {
	*ecdsa.PrivateKey
}

func (k *vapidKey) encodeToDER(isPrivate bool) (encoded []byte, err error) {
	var derEncoded []byte

	if isPrivate {
		if derEncoded, err = x509.MarshalECPrivateKey(k.PrivateKey); err != nil {
			log.Println(err)
			return nil, fmt.Errorf("failed to encode ECDSA private key to DER format")
		}
	} else {
		if derEncoded, err = x509.MarshalPKIXPublicKey(&k.PublicKey); err != nil {
			log.Println(err)
			return nil, fmt.Errorf("failed to encode ECDSA public key to DER format")
		}
	}

	return derEncoded, err
}

func (k *vapidKey) EncodeToPEM(isPrivate bool) (p string, err error) {
	var encoded []byte

	if isPrivate {
		if encoded, err = k.encodeToDER(true); err != nil {
			return
		}

		encoded = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encoded})
	} else {
		if encoded, err = k.encodeToDER(false); err != nil {
			return
		}

		encoded = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})
	}

	return string(encoded), nil
}

func (k *vapidKey) String() (s string) {
	ecdhPublicKey, err := k.PublicKey.ECDH()

	if err != nil {
		log.Println(err)
		return
	}

	return base64.RawURLEncoding.EncodeToString(ecdhPublicKey.Bytes())
}

func parseECPrivateKey(pemBlock *pem.Block) (privateKey *ecdsa.PrivateKey, err error) {
	if privateKey, err = x509.ParseECPrivateKey(pemBlock.Bytes); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to parse SEC1 encoded PEM")
	}

	return
}

func parsePKCS8PrivateKey(pemBlock *pem.Block) (privateKey *ecdsa.PrivateKey, err error) {
	var generic any

	if generic, err = x509.ParsePKCS8PrivateKey(pemBlock.Bytes); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to parse PKCS#8 PEM")
	}

	var ok bool

	if privateKey, ok = generic.(*ecdsa.PrivateKey); !ok {
		return nil, fmt.Errorf("parsed PKCS#8 PEM is not of type ECDSA private key")
	}

	return
}

func validateCurve(privateKey *ecdsa.PrivateKey) (err error) {
	curveParams := privateKey.Curve.Params()

	if curveParams.BitSize != CURVE_BIT_SIZE {
		return fmt.Errorf("curve has wrong bit size: %v, expected %v", curveParams.BitSize, CURVE_BIT_SIZE)
	}

	return
}

func DecodeFromPEM(pemEncoded string) (k *vapidKey, err error) {
	pemBlock, _ := pem.Decode([]byte(pemEncoded))

	if pemBlock == nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey := new(ecdsa.PrivateKey)

	switch pemBlock.Type {
	case "EC PRIVATE KEY":
		privateKey, err = parseECPrivateKey(pemBlock)
	case "PRIVATE KEY":
		privateKey, err = parsePKCS8PrivateKey(pemBlock)
	default:
		return nil, fmt.Errorf("only SEC1- or PKCS#8 encoded PEMs are supported")
	}

	if err == nil {
		err = validateCurve(privateKey)
	}

	if err != nil {
		return nil, err
	}

	return &vapidKey{
		privateKey,
	}, nil
}

func GenerateVapidKey() (*vapidKey, error) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to generate ECDSA P-256 private key")
	}

	return &vapidKey{
		private,
	}, nil
}
