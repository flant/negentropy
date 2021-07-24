package multipass

import (
	"context"
	"fmt"

	"github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn"
	authnjwt "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn/jwt"
	auth_usecase "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

type Authenticator struct {
	JwtValidator *jwt.Validator

	MultipassService *auth_usecase.Multipass

	AuthMethod *model.AuthMethod
	AuthSource *model.AuthSource

	Logger hclog.Logger
}

func (a *Authenticator) Authenticate(ctx context.Context, d *framework.FieldData) (*authn.Result, error) {
	a.Logger.Debug("Start authn multipass")

	authenticator := &authnjwt.Authenticator{
		AuthMethod:   a.AuthMethod,
		Logger:       a.Logger.Named("JWT"),
		AuthSource:   a.AuthSource,
		JwtValidator: a.JwtValidator,
	}

	res, err := authenticator.Authenticate(ctx, d)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Try to get jti from claims %s", res.UUID))
	jtiFromTokenRaw := authnjwt.GetClaim(a.Logger, res.Claims, "jti")
	if jtiFromTokenRaw == nil {
		return nil, fmt.Errorf("not found jti from token")
	}

	jtiFromToken, ok := jtiFromTokenRaw.(string)
	if !ok {
		return nil, fmt.Errorf("jti must be string")
	}

	a.Logger.Debug(fmt.Sprintf("Try to get multipass with its gen %s", res.UUID))
	multipass, multipassGen, err := a.MultipassService.GetWithGeneration(res.UUID)
	if err != nil {
		return nil, err
	}

	if multipass.Salt == "" {
		a.Logger.Error(fmt.Sprintf("Got empty salt %s", res.UUID))
		return nil, fmt.Errorf("jti is not valid")
	}

	jti := usecase.TokenJTI{
		Generation: multipassGen.GenerationNumber,
		SecretSalt: multipass.Salt,
	}.Hash()

	a.Logger.Debug(fmt.Sprintf("Verify jti %s", res.UUID))

	if jti != jtiFromToken {
		a.Logger.Error(fmt.Sprintf("Incorrect jti got=%s need=%s", jtiFromToken, jti))
		return nil, fmt.Errorf("jti is not valid")
	}

	a.Logger.Debug(fmt.Sprintf("Found multipass owner %s", multipass.OwnerUUID))

	return &authn.Result{
		UUID:         multipass.OwnerUUID,
		Metadata:     map[string]string{},
		GroupAliases: make([]string, 0),
	}, nil
}
