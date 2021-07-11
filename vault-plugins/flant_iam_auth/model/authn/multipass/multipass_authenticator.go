package multipass

import (
	"context"
	"fmt"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/hashicorp/go-hclog"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn"
	gojwt "github.com/golang-jwt/jwt"
	"github.com/hashicorp/vault/sdk/framework"
)

type Authenticator struct {
	// todo use validator
	//JwtValidator *jwt.Validator
	AuthSource     *model.AuthSource
	MultipassRepo  *iam.MultipassRepository
	Logger       hclog.Logger
}

func (a *Authenticator) Authenticate(ctx context.Context, d *framework.FieldData) (*authn.Result, error) {
	token := d.Get("jwt").(string)
	if len(token) == 0 {
		return nil, fmt.Errorf("missing token")
	}

	p := gojwt.Parser{SkipClaimsValidation: true }
	claims := gojwt.MapClaims{}
	_, _, err := p.ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}

	if !claims.VerifyIssuer(a.AuthSource.BoundIssuer, true) {
		return nil, fmt.Errorf("incorrect issuer")
	}

	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return nil, fmt.Errorf("incorrect expiration")
	}

	uuidMultipassRaw, ok := claims["uuid"]
	if !ok {
		return nil, fmt.Errorf("not found uuid")
	}

	uuidMultipass, ok := uuidMultipassRaw.(string)

	multipass, err := a.MultipassRepo.GetByID(uuidMultipass)
	if err != nil {
		return nil, err
	}

	if multipass == nil {
		return nil, fmt.Errorf("not found multipass")
	}

	return &authn.Result{
		UUID: multipass.OwnerUUID,
		Metadata: map[string]string{},
		GroupAliases: make([]string, 0),
	}, nil
}
