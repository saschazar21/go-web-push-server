package vapid

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/saschazar21/go-web-push-server/utils"
)

type vapidClaims struct {
	Sub string       `json:"sub" validate:"mailto"`
	Aud string       `json:"aud" validate:"origin"`
	Exp *utils.Epoch `json:"exp" validate:"epoch-gt-now"`
}

func (c *vapidClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.Exp == nil {
		return &jwt.NumericDate{
			Time: time.Unix(0, 0),
		}, nil
	}

	return &jwt.NumericDate{
		Time: time.Time(*c.Exp),
	}, nil
}

func (c *vapidClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return &jwt.NumericDate{
		// Time: time.Now().UTC(),
		Time: time.Unix(0, 0),
	}, nil
}

func (c *vapidClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return &jwt.NumericDate{
		Time: time.Unix(0, 0),
	}, nil
}

func (c *vapidClaims) GetIssuer() (string, error) {
	return "", nil
}

func (c *vapidClaims) GetSubject() (string, error) {
	return c.Sub, nil
}

func (c *vapidClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{c.Aud}, nil
}

func (c *vapidClaims) Validate() (err error) {
	return utils.CustomValidateStruct(c)
}
