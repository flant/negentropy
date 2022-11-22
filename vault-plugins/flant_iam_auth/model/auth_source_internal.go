package model

import (
	"crypto"

	"github.com/hashicorp/cap/jwt"

	flantjwt "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

const (
	MultipassSourceName          = "_internal_multipass"
	ServiceAccountPassSourceName = "_internal_service_account_pass"
)

func GetMultipassSource() *AuthSource {
	return &AuthSource{
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
		Name:                 ServiceAccountPassSourceName,
		EntityAliasName:      EntityAliasNameFullIdentifier,
		AllowServiceAccounts: true,
		OnlyServiceAccounts:  true,
	}
}
