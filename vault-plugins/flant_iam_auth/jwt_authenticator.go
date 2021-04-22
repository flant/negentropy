package jwtauth

import (
	"context"
	"fmt"
	"github.com/hashicorp/cap/jwt"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type JwtAuthenticator struct {
	jwtValidator *jwt.Validator
	authMethod   *authMethodConfig
	authSource   *authSource
	logger       log.Logger
	methodName   string
}

func (a *JwtAuthenticator) Auth(ctx context.Context, d *framework.FieldData) (*logical.Auth, error) {
	token := d.Get("jwt").(string)
	if len(token) == 0 {
		return nil, fmt.Errorf("missing token")
	}

	// Validate JWT supported algorithms if they've been provided. Otherwise,
	// ensure that the signing algorithm is a member of the supported set.
	signingAlgorithms := toAlg(a.authSource.JWTSupportedAlgs)
	if len(signingAlgorithms) == 0 {
		signingAlgorithms = []jwt.Alg{
			jwt.RS256, jwt.RS384, jwt.RS512, jwt.ES256, jwt.ES384,
			jwt.ES512, jwt.PS256, jwt.PS384, jwt.PS512, jwt.EdDSA,
		}
	}

	// Set expected claims values to assert on the JWT
	expected := jwt.Expected{
		Issuer:            a.authSource.BoundIssuer,
		Subject:           a.authMethod.BoundSubject,
		Audiences:         a.authMethod.BoundAudiences,
		SigningAlgorithms: signingAlgorithms,
		NotBeforeLeeway:   a.authMethod.NotBeforeLeeway,
		ExpirationLeeway:  a.authMethod.ExpirationLeeway,
		ClockSkewLeeway:   a.authMethod.ClockSkewLeeway,
	}

	// Validate the JWT by verifying its signature and asserting expected claims values
	allClaims, err := a.jwtValidator.Validate(ctx, token, expected)
	if err != nil {
		return nil, fmt.Errorf("error validating token: %s", err.Error())
	}

	// If there are no bound audiences for the authMethodConfig, then the existence of any audience
	// in the audience claim should result in an error.
	aud, ok := getClaim(a.logger, allClaims, "aud").([]interface{})
	if ok && len(aud) > 0 && len(a.authMethod.BoundAudiences) == 0 {
		return nil, fmt.Errorf("audience claim found in JWT but no audiences bound to the method")
	}

	alias, groupAliases, err := createIdentity(a.logger, allClaims, a.authMethod, nil)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	if err := validateBoundClaims(a.logger, a.authMethod.BoundClaimsType, a.authMethod.BoundClaims, allClaims); err != nil {
		return nil, fmt.Errorf("error validating claims: %s", err.Error())
	}

	tokenMetadata := map[string]string{"flantIamAuthMethod": a.methodName}
	for k, v := range alias.Metadata {
		tokenMetadata[k] = v
	}

	return &logical.Auth{
		DisplayName:  alias.Name,
		Alias:        alias,
		GroupAliases: groupAliases,
		InternalData: map[string]interface{}{},
		Metadata:     tokenMetadata,
	}, nil
}
