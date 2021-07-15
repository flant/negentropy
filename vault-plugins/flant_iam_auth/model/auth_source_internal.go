package model

import (
	"github.com/hashicorp/cap/jwt"

	flantjwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
)

const (
	MultipassSourceUUID = "4554696c-e53b-11eb-bf72-a7d3a66da383"
	MultipassSourceName = "_internal_multipass"
)

func GetMultipassSource() *AuthSource {
	return &AuthSource{
		UUID:                 MultipassSourceUUID,
		Name:                 MultipassSourceName,
		JWTSupportedAlgs:     []string{string(jwt.EdDSA)},
		EntityAliasName:      EntityAliasNameFullIdentifier,
		AllowServiceAccounts: true,
	}
}

func GetMultipassSourceForLogin(jwtConf *flantjwt.Config, keys []string) *AuthSource {
	source := GetMultipassSource()
	source.JWTValidationPubKeys = keys
	source.BoundIssuer = jwtConf.Issuer

	return source
}
