package model

import (
	"crypto"

	"github.com/hashicorp/cap/jwt"

	flantjwt "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

const (
	MultipassSourceUUID          = "4554696c-e53b-11eb-bf72-a7d3a66da383"
	MultipassSourceName          = "_internal_multipass"
	ServiceAccountPassSourceUUID = "be4ef289-fe1a-4d91-ad7b-a33c0d792102"
	ServiceAccountPassSourceName = "_internal_service_account_pass"
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

func GetMultipassSourceForLogin(jwtConf *flantjwt.Config, keys []crypto.PublicKey) *AuthSource {
	source := GetMultipassSource()
	source.ParsedJWTPubKeys = keys
	source.BoundIssuer = jwtConf.Issuer
	return source
}

func GetServiceAccountPassSource() *AuthSource {
	return &AuthSource{
		UUID:                 ServiceAccountPassSourceUUID,
		Name:                 ServiceAccountPassSourceName,
		EntityAliasName:      EntityAliasNameFullIdentifier,
		AllowServiceAccounts: true,
		OnlyServiceAccounts:  true,
	}
}
