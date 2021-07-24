package multipass

import (
	"context"
	"fmt"

	"github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
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

	multipass, err := a.verifyMultipass(res.UUID, jtiFromToken)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Found multipass owner %s", multipass.OwnerUUID))

	return &authn.Result{
		UUID:         multipass.OwnerUUID,
		Metadata:     map[string]string{},
		GroupAliases: make([]string, 0),
		InternalData: map[string]interface{}{
			"multipass": map[string]interface{}{
				"multipass_id": multipass.UUID,
				"jti":          jtiFromToken,
			},
		},
	}, nil
}

func (a *Authenticator) CanRenew(vaultAuth *logical.Auth) (bool, error) {
	rawMpAuth, ok := vaultAuth.InternalData["multipass"]
	if !ok {
		return false, fmt.Errorf("not found multipass data")
	}

	mpAuth, ok := rawMpAuth.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("not cast multipass data")
	}

	_, err := a.verifyMultipass(mpAuth["multipass_id"].(string), mpAuth["jti"].(string))
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *Authenticator) verifyMultipass(uuid, jtiExpected string) (*iam.Multipass, error) {
	a.Logger.Debug(fmt.Sprintf("Try to get multipass with its gen %s", uuid))
	multipass, multipassGen, err := a.MultipassService.GetWithGeneration(uuid)
	if err != nil {
		return nil, err
	}

	if multipass.Salt == "" {
		a.Logger.Error(fmt.Sprintf("Got empty salt %s", uuid))
		return nil, fmt.Errorf("jti is not valid")
	}

	jti := usecase.TokenJTI{
		Generation: multipassGen.GenerationNumber,
		SecretSalt: multipass.Salt,
	}.Hash()

	a.Logger.Debug(fmt.Sprintf("Verify jti %s", uuid))

	if jti != jtiExpected {
		a.Logger.Error(fmt.Sprintf("Incorrect jti got=%s need=%s", jtiExpected, jti))
		return nil, fmt.Errorf("jti is not valid")
	}

	return multipass, nil
}
