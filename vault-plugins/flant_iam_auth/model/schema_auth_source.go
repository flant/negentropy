package model

import (
	"crypto"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/certutil"
	"github.com/hashicorp/vault/sdk/helper/strutil"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const (
	AuthSourceType = "auth_source" // also, memdb schema name
)

const (
	AuthSourceStaticKeys = iota
	AuthSourceJWKS
	AuthSourceOIDCDiscovery
	AuthSourceOIDCFlow
	Unconfigured
)

const (
	ResponseTypeCode     = "code"      // Authorization code flow
	ResponseTypeIDToken  = "id_token"  // ID Token for form post
	ResponseModeQuery    = "query"     // Response as a redirect with query parameters
	ResponseModeFormPost = "form_post" // Response as an HTML Form
)

const (
	EntityAliasNameEmail          = "email"
	EntityAliasNameFullIdentifier = "full_identifier"
	EntityAliasNameUUID           = "uuid"
)

type AuthSource struct {
	UUID                 string   `json:"uuid"` // ID
	Name                 string   `json:"name"`
	OIDCDiscoveryURL     string   `json:"oidc_discovery_url"`
	OIDCDiscoveryCAPEM   string   `json:"oidc_discovery_ca_pem"`
	OIDCClientID         string   `json:"oidc_client_id"`
	OIDCClientSecret     string   `json:"oidc_client_secret"`
	OIDCResponseMode     string   `json:"oidc_response_mode"`
	OIDCResponseTypes    []string `json:"oidc_response_types"`
	JWKSURL              string   `json:"jwks_url"`
	JWKSCAPEM            string   `json:"jwks_ca_pem"`
	JWTValidationPubKeys []string `json:"jwt_validation_pubkeys"`
	JWTSupportedAlgs     []string `json:"jwt_supported_algs"`
	BoundIssuer          string   `json:"bound_issuer"`
	DefaultRole          string   `json:"default_role"`
	NamespaceInState     bool     `json:"namespace_in_state"`
	EntityAliasName      string   `json:"entity_alias_name"`
	AllowServiceAccounts bool     `json:"allow_service_accounts"`

	ParsedJWTPubKeys []crypto.PublicKey `json:"-"`
}

func (s *AuthSource) ObjType() string {
	return AuthSourceType
}

func (s *AuthSource) ObjId() string {
	return s.UUID
}

func (s *AuthSource) AllowForSA() bool {
	return s.AllowServiceAccounts && s.EntityAliasName != EntityAliasNameEmail
}

func (s *AuthSource) NameForServiceAccount(sa *iam.ServiceAccount) (string, error) {
	var name string
	switch s.EntityAliasName {
	case EntityAliasNameFullIdentifier:
		name = sa.FullIdentifier
	case EntityAliasNameUUID:
		name = sa.UUID
	default:
		return "", fmt.Errorf("incorrect source entity alias name %s", s.EntityAliasName)
	}

	return name, nil
}

func (s *AuthSource) NameForUser(user *iam.User) (string, error) {
	var name string
	switch s.EntityAliasName {
	case EntityAliasNameEmail:
		name = user.Email
	case EntityAliasNameFullIdentifier:
		name = user.FullIdentifier
	case EntityAliasNameUUID:
		name = user.UUID
	default:
		return "", fmt.Errorf("incorrect source entity alias name %s", s.EntityAliasName)
	}

	return name, nil
}

func (s *AuthSource) PopulatePubKeys() error {
	for _, v := range s.JWTValidationPubKeys {
		key, err := certutil.ParsePublicKeyPEM([]byte(v))
		if err != nil {
			return errwrap.Wrapf("error parsing public key: {{err}}", err)
		}
		s.ParsedJWTPubKeys = append(s.ParsedJWTPubKeys, key)
	}

	return nil
}

// AuthType classifies the authorization type/flow based on config parameters.
func (s *AuthSource) AuthType() int {
	switch {
	case len(s.ParsedJWTPubKeys) > 0:
		return AuthSourceStaticKeys
	case s.JWKSURL != "":
		return AuthSourceJWKS
	case s.OIDCDiscoveryURL != "":
		if s.OIDCClientID != "" && s.OIDCClientSecret != "" {
			return AuthSourceOIDCFlow
		}
		return AuthSourceOIDCDiscovery
	}

	return Unconfigured
}

// HasType returns whether the list of response types includes the requested
// type. The default type is 'code' so that special case is handled as well.
func (s *AuthSource) HasType(t string) bool {
	if len(s.OIDCResponseTypes) == 0 && t == ResponseTypeCode {
		return true
	}

	return strutil.StrListContains(s.OIDCResponseTypes, t)
}

func AuthSourceSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			AuthSourceType: {
				Name: AuthSourceType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByName: {
						Name:   ByName,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}
