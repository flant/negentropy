package jwt

import (
	"context"
	"fmt"

	"github.com/hashicorp/cap/jwt"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
)

type JwtValidator interface {
	Validate(ctx context.Context, token string, expected jwt.Expected) (allClaims map[string]interface{}, err error)
}

type Authenticator struct {
	JwtValidator JwtValidator
	AuthMethod   *model.AuthMethod
	AuthSource   *model.AuthSource
	Logger       log.Logger
}

func (a *Authenticator) Authenticate(ctx context.Context, d *framework.FieldData) (*authn2.Result, error) {
	token := d.Get("jwt").(string)
	if len(token) == 0 {
		return nil, fmt.Errorf("missing token")
	}

	a.Logger.Debug("Start validate jwt")
	// Validate JWT supported algorithms if they've been provided. Otherwise,
	// ensure that the signing algorithm is a member of the supported set.
	signingAlgorithms := ToAlg(a.AuthSource.JWTSupportedAlgs)
	if len(signingAlgorithms) == 0 {
		signingAlgorithms = []jwt.Alg{
			jwt.RS256, jwt.RS384, jwt.RS512, jwt.ES256, jwt.ES384,
			jwt.ES512, jwt.PS256, jwt.PS384, jwt.PS512, jwt.EdDSA,
		}
	}

	a.Logger.Debug("Got jwt supported algs")

	// Set expected claims values to assert on the JWT
	expected := jwt.Expected{
		Issuer:            a.AuthSource.BoundIssuer,
		Subject:           a.AuthMethod.BoundSubject,
		Audiences:         a.AuthMethod.BoundAudiences,
		SigningAlgorithms: signingAlgorithms,
		NotBeforeLeeway:   a.AuthMethod.NotBeforeLeeway,
		ExpirationLeeway:  a.AuthMethod.ExpirationLeeway,
		ClockSkewLeeway:   a.AuthMethod.ClockSkewLeeway,
	}

	a.Logger.Debug("Start validate signature")
	// Validate the JWT by verifying its signature and asserting expected claims values
	allClaims, err := a.JwtValidator.Validate(ctx, token, expected)
	if err != nil {
		return nil, fmt.Errorf("error validating token: %s", err.Error())
	}

	a.Logger.Debug("Get claims")

	// If there are no bound audiences for the authMethodConfig, then the existence of any audience
	// in the audience claim should result in an error.
	aud, ok := GetClaim(a.Logger, allClaims, "aud").([]interface{})
	if ok && len(aud) > 0 && len(a.AuthMethod.BoundAudiences) == 0 {
		return nil, fmt.Errorf("audience claim found in JWT but no audiences bound to the method")
	}

	alias, groupAliases, err := CreateIdentity(a.Logger, allClaims, a.AuthMethod, nil)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	if err := ValidateBoundClaims(a.Logger, a.AuthMethod.BoundClaimsType, a.AuthMethod.BoundClaims, allClaims); err != nil {
		return nil, fmt.Errorf("error validating claims: %s", err.Error())
	}

	return &authn2.Result{
		UUID:      alias.Name,
		ModelType: "", // its unknown

		Metadata:     alias.Metadata,
		GroupAliases: groupAliases,

		Claims: allClaims,
	}, nil
}

func (a *Authenticator) CanRenew(vaultAuth *logical.Auth) (bool, error) {
	// TODO may be add logiс?
	return true, nil
}
