package webpush

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ERROR_MSG = "TestVapidKey err = %v, wantErr = %v"

func TestVapidKey(t *testing.T) {

	t.Run("creates vapidKey", func(t *testing.T) {
		var k *vapidKey
		var err error

		if k, err = GenerateVapidKey(); err != nil {
			t.Errorf(ERROR_MSG, err, nil)
		}

		assert.NotNil(t, k)

		assert.Greater(t, len(k.String()), 0)
	})

	type test struct {
		name    string
		pem     string
		wantErr bool
	}

	tests := []test{
		{
			"pkcs#8 pem",
			`
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgMyZbxzTTxc90kutd
Or/pkwmUQF3fYHHJH91+/mvqxbWhRANCAAStcsbIPfQzgAYbQkVQhNGiIul5T5dD
smDlca4rkc2LLpSH8a+2rM47h53KpAjMKGmf3mmmLRTBNPhnGvM1w+TP
-----END PRIVATE KEY-----
			`,
			false,
		}, {
			"sec1 pem",
			`
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIJhmgSYljUqG0+VhEtyEaAU+SbEv+9uqxSD100UXO7M9oAoGCCqGSM49
AwEHoUQDQgAETrAUmBWDPOhyQV7j3WlO/n3+r91lgcWCrpSBC7/Oo1ezBAe2yvLQ
nKagi/21wOhk82wsaxy6VuOOmIoW7SNjuQ==
-----END EC PRIVATE KEY-----
			`,
			false,
		}, {
			"es256 pkcs#8 public key pem",
			`
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEJ4oQsHmVB9aBaH+i1DYIDUqYpqX9
+f6iYTi0a5vD2SnjoCPzMO5S02nuHq5QHeUESJbbCtR1YTdPkF6viWyXSw==
-----END PUBLIC KEY-----
			`,
			true,
		}, {
			"rsa pkcs#8 pem",
			`
-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAKA1vk8fg3D6MrD8
RfKRn3wdgb02DdJcIzk1jA4b+saFfHt3nOT3OUDhdKpomr2abhZoEIWRKHUTMTnn
2nJ2jE2YYv+AxH2tjw2vy0SieR9Y40MaB9dObWijK5uTtZKIaQgayBlLrueVV3iC
B4CQaMcT8gk4fxkilf0aSIGht2JHAgMBAAECgYA6hAaznZ4DqM7VB/+AXqHy0lAt
zM11lQOkhKNYD+4jjmPuML0UgBvgT7it+TDzqbEl6/KE5oTxZgYn0UBfaF9L7nrD
lTJZzjn5C49cT/w5C1191csmB32tzBOWt/9hNCdSCnNpUHh3FzKGHtey0cg4qCJW
Aducj7JeDSmNx90XQQJBAOwYj2LYjB8fEoXOSRkJqBhZ22m9xuNYwPUvR21MJJAR
qVwSiz61NPGsY0A7RbpLCXsm39/mWgo9pTFRmhJw2eECQQCtt2fvkkSl+pp6VV2B
9vqDV08IwaIbG8kKBT1Xs+1OgNIiCYAbIzgwK1H9EIpupOVjX5pds78HEue3bj9s
VlEnAkAIq9JAWCG1VufQQEZRBBjHZC150b2HRhA4MRdXfU9udyeYORoiIHekVKeE
iWjDMdRUUJYyW/x8mc0CZbPZ74khAkAv0sAASiov72+7oeieMNoCcnTFmlkAUYPl
CFA85sG7zOcMi8UCs41yZVqq6nTRxP+JffZHOYarcd7stqMrNhAdAkBy7Rx4bBMM
O/6SaMLB5PSyTLPAlKKLZPsym725cyBzbtZoP55gzZgBEAxYdJZpWo51Pc5Qzh6G
DL41vNpJkhjV
-----END PRIVATE KEY-----
			`,
			true,
		}, {
			"es512 pkcs#8 pem",
			`
-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIAh2rYgxMSNL+mcD1p
5N61esfd+OBTI+6dhr9ElQg18ewwNZjR/6U0c7FGRMN+RrDyFC8fIB/W1o8y0Woo
hPGofGyhgYkDgYYABAA5s+cDlfyvIJEuaMtP219pxn5jIIl+nL8prUlnspp5pYEu
v0k7GjIn5E8S6Mev0N1EYQ02qMvOVpvg26Ma6yrPqQEpDiqVuK4lkd5MkI6pElWn
BLETx9LgPeym7dbEnvLtWGZ/IdOjsPQ4vDRGC+fGgWFVKflxPn4am9w6j5UBGF4l
kg==
-----END PRIVATE KEY-----
			`,
			true,
		}, {
			"empty pkcs#8 pem",
			`
-----BEGIN PRIVATE KEY-----
-----END PRIVATE KEY-----
			`,
			true,
		}, {
			"empty sec1 pem",
			`
-----BEGIN EC PRIVATE KEY-----
-----END EC PRIVATE KEY-----
			`,
			true,
		}, {
			"malformatted pem",
			`
malformatted
			`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			k := new(vapidKey)

			if k, err = DecodeFromPEM(tt.pem); (err != nil) != tt.wantErr {
				t.Errorf("TestVapidKey err = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {
				var enc string

				if enc, err = k.EncodeToPEM(false); err != nil {
					t.Errorf("TestVapidKey err = %v, wantErr = %v", err, false)
				}

				assert.NotEmpty(t, enc)

				if enc, err = k.EncodeToPEM(true); err != nil {
					t.Errorf("TestVapidKey err = %v, wantErr = %v", err, false)
				}

				assert.NotEmpty(t, enc)

				log.Println(enc)
			}
		})
	}
}
