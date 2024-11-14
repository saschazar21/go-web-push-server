package webpush

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVapid(t *testing.T) {
	t.Setenv(VAPID_EXPIRY_DURATION_ENV, "300")
	t.Setenv(VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
	`)
	t.Setenv(VAPID_SUBJECT_ENV, "test@example.com")

	type test struct {
		name    string
		env     [][]string
		aud     string
		wantErr bool
	}

	tests := []test{
		{
			"validates",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com",
			false,
		},
		{
			"fails to validate aud",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com/asdf/123",
			true,
		},
		{
			"validates invalid exp env",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "-300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com",
			false,
		},
		{
			"validates invalid exp env",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "k"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com",
			false,
		},
		{
			"fails to validate invalid private key",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDHw+AglgGpF6sv
c/tW89lcN0bGb2EHw+dP/74OW4X31N+s7l/DbE0tYUQSge2kHDnz2Ctzrp2RxSVy
ddsHOvM0Apqgqi0vYE8ljlJ48kqWZr55CI2I+tLdKYz4lIBeBqH2NVc8WwoQ2QHq
zDsBfgXOX2iHU1IJWg1BMcUpXqfPDDTWILnX2U9vg0U3rYIsRoDsafyxIZwEfjRD
Qg2N/gtILPlg8ArPirooI1khVQ7QW7WHB6JSZjVnldw80C2rEzPoe88sLCa09lFN
eNk4qBkBWUViTQ29ZV+BMeJ1+9KGRDNUowQH1jNvDv3N1H9Fzy0mTKdSe05H/oBR
fPK8EvcXAgMBAAECggEAWkpEw8W5U2mwxH56JEeMP3t2gFs4Mo/PvZ9cklW4vBcZ
0Cpf207YpUG4yFq0hAAEC5xxq1RJwOioL89oI6D36tKgfCzexnKT42gsC6GLp+Yh
gkgk3Lxt0Wul3XcVfCooS0W5u7x0VMAY9zy/EMIasrf54Wx+AF8U7ZomwLeZRmGz
poyKNJJIKBSrtKbkGVnr5zD90Z2Cr5z/CzzEpiq3DN0eV2Bo6OhMbf5q6taZ27nt
+IVUu4GrEvEZvoSNI2hFjr35vS796HAu7XqUX4yXyuLeHv4TsGtmR0CKI3UmvWX8
TB+2K7nEYtxTqLbmPBDwxl3AOiZOzAx5LXmKi8CKwQKBgQDxTdew0VF5LEui8Nss
wz+zfcFxEh2aDWpht7vD0icV85XmMnsB680Ypbo5idAwyrOSpSVlYSgl6qQrwC7P
sTDTUv2l/Ulsh01cXuexbRXludmAAuhV7djVGYwr5iL2XaoiReHgtW5k+TytIHsJ
GBSTuJ4+yY0s9DNPJ3ZZH0g11wKBgQDT7mbjXHCx7mTZKncUu6SlAJTXOCjyZ0rw
FAufni0I0pcSulCCdGzaIJxN5lmpkHw/o+k4CgnKbiR81sj19PaWeBHPxsTd+KoH
knTg/uuHe2qxMr7HUcsHEMERMuQ+nuamOVkp02kkCTB9q0HrleXxM7iTRbNtZhHr
VHZWS2SgwQKBgE8Jai6WQRNpeNTEA2YkBcdq12OLxXpiDog3QB8hxH+iK2Uc/8Ff
VOxPzDFwfGqe2jacNSWBrz7MHj3eUvbgWNe/BSnLTrNnleU9iLJKwrNeLmmJikQr
Bay3E3yFgsojX8ieDyAlDSWxpTgnvWT7KDJCdEKojb89tVil2lPStTo9AoGBAM4i
596j7lWTPHJitJrs/PMlQqCn1mQZBjHIPZoO31zigOFNabvKBIqSB5ZZxMKCb+fy
xYilcup8AW+P9r4Ne7/Vn/WKL7h8At4EnTyvl2YbLCaY5im3LBR+Plw9NPaX1l6+
DzT4lh7f9VN2vVKpZZQbq59Lv39cNXfBmqzK/mDBAoGAU6AyYb+fFYoZUQM+scFY
N1k0Ay92mVqZMcpqeN2737gNNbaB4zveF4cMx7cKBG4RVkNpz2A1UM22BQ5uVD3P
LdXbSFJdgJtYU/m7aAZ0QD74RQG+pVXqxd1CFV3F6FA2dybAn9t44uXkaSHRwQts
kgAzzkxuofY+ylkjO9rx44k=
-----END PRIVATE KEY-----
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com",
			true,
		},
		{
			"fails to validate unsupported private key",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
asdf
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "test@example.com"},
			},
			"https://test.example.com",
			true,
		},
		{
			"fails to validate invalid subject",
			[][]string{
				{VAPID_EXPIRY_DURATION_ENV, "300"},
				{VAPID_PRIVATE_KEY_ENV, `
-----BEGIN PRIVATE KEY-----
MEECAQAwEwYHKoZIzj0CAQYIKoZIzj0DAQcEJzAlAgEBBCCFZOAAzpzloIIUnRsT
MK468C66gOKehSQqxUQ8+HCI/g==
-----END PRIVATE KEY-----	
				`},
				{VAPID_SUBJECT_ENV, "swift@invalid"},
			},
			"https://test.example.com",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pair := range tt.env {
				t.Setenv(pair[0], pair[1])
			}

			jwt, key, err := NewVAPID(tt.aud)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestVapid err = %v, wantErr = %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				assert.NotEmpty(t, jwt)
				assert.NotEmpty(t, key)

				fmt.Printf("%v\n%v\n", jwt, key)
			}
		})
	}
}
