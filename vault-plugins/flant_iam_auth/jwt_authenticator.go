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

	alias, groupAliases, err := a.createIdentity(allClaims, a.authMethod)
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

// createIdentity creates an alias and set of groups aliases based on the authMethodConfig
// definition and received claims.
func (a *JwtAuthenticator) createIdentity(allClaims map[string]interface{}, authMethod *authMethodConfig) (*logical.Alias, []*logical.Alias, error) {
	userClaimRaw, ok := allClaims[authMethod.UserClaim]
	if !ok {
		return nil, nil, fmt.Errorf("claim %q not found in token", authMethod.UserClaim)
	}
	userName, ok := userClaimRaw.(string)
	if !ok {
		return nil, nil, fmt.Errorf("claim %q could not be converted to string", authMethod.UserClaim)
	}

	metadata, err := extractMetadata(a.logger, allClaims, authMethod.ClaimMappings)
	if err != nil {
		return nil, nil, err
	}

	alias := &logical.Alias{
		Name:     userName,
		Metadata: metadata,
	}

	var groupAliases []*logical.Alias

	if authMethod.GroupsClaim == "" {
		return alias, groupAliases, nil
	}

	groupsClaimRaw := getClaim(a.logger, allClaims, authMethod.GroupsClaim)

	if groupsClaimRaw == nil {
		return nil, nil, fmt.Errorf("%q claim not found in token", authMethod.GroupsClaim)
	}

	groups, ok := normalizeList(groupsClaimRaw)

	if !ok {
		return nil, nil, fmt.Errorf("%q claim could not be converted to string list", authMethod.GroupsClaim)
	}
	for _, groupRaw := range groups {
		group, ok := groupRaw.(string)
		if !ok {
			return nil, nil, fmt.Errorf("value %v in groups claim could not be parsed as string", groupRaw)
		}
		if group == "" {
			continue
		}
		groupAliases = append(groupAliases, &logical.Alias{
			Name: group,
		})
	}

	return alias, groupAliases, nil
}
