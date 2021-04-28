package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/helper/tokenutil"
)

const (
	AuthMethodType = "auth_method" // also, memdb schema name
)

const (
	ClaimDefaultLeeway    = 150
	BoundClaimsTypeString = "string"
	BoundClaimsTypeGlob   = "glob"
)

const (
	MethodTypeJWT        = "jwt"
	MethodTypeOIDC       = "oidc"
	MethodTypeOwn        = "jwt_own"
	MethodTypeSAPassword = "service_account_password"
)

func AuthMethodSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			AuthMethodType: {
				Name: AuthMethodType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByName: {
						Name: ByName,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

type AuthMethod struct {
	tokenutil.TokenParams

	UUID       string `json:"uuid"` // ID
	Name       string `json:"name"`
	MethodType string `json:"method_type"`
	Source     string `json:"source"`

	// Duration of leeway for expiration to account for clock skew
	ExpirationLeeway time.Duration `json:"expiration_leeway"`

	// Duration of leeway for not before to account for clock skew
	NotBeforeLeeway time.Duration `json:"not_before_leeway"`

	// Duration of leeway for all claims to account for clock skew
	ClockSkewLeeway time.Duration `json:"clock_skew_leeway"`

	// Role binding properties
	BoundAudiences      []string               `json:"bound_audiences"`
	BoundSubject        string                 `json:"bound_subject"`
	BoundClaimsType     string                 `json:"bound_claims_type"`
	BoundClaims         map[string]interface{} `json:"bound_claims"`
	ClaimMappings       map[string]string      `json:"claim_mappings"`
	UserClaim           string                 `json:"user_claim"`
	GroupsClaim         string                 `json:"groups_claim"`
	OIDCScopes          []string               `json:"oidc_scopes"`
	AllowedRedirectURIs []string               `json:"allowed_redirect_uris"`
	VerboseOIDCLogging  bool                   `json:"verbose_oidc_logging"`
	MaxAge              time.Duration          `json:"max_age"`
}

func (p *AuthMethod) ObjType() string {
	return AuthMethodType
}

func (p *AuthMethod) ObjId() string {
	return p.UUID
}

func (p *AuthMethod) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(p)
}

func (p *AuthMethod) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, p)
	return err
}
