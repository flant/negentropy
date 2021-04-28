package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	ServiceAccountType = "service_account" // also, memdb schema name

)

type ServiceAccountObjectType string

func ServiceAccountSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServiceAccountType: {
				Name: ServiceAccountType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

type ServiceAccount struct {
	UUID           string        `json:"uuid"` // ID
	TenantUUID     string        `json:"tenant_uuid"`
	Version        string        `json:"resource_version"`
	BuiltinType    string        `json:"-"`
	Identifier     string        `json:"identifier"`
	FullIdentifier string        `json:"full_identifier"`
	CIDRs          []string      `json:"allowed_cidrs"`
	TokenTTL       time.Duration `json:"token_ttl"`
	TokenMaxTTL    time.Duration `json:"token_max_ttl"`
}

func (u *ServiceAccount) ObjType() string {
	return ServiceAccountType
}

func (u *ServiceAccount) ObjId() string {
	return u.UUID
}

func (u *ServiceAccount) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *ServiceAccount) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

// generic: <identifier>@serviceaccount.<tenant_identifier>
// builtin: <identifier>@<builtin_serviceaccount_type>.serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(sa *ServiceAccount, tenant *Tenant) string {
	name := sa.Identifier
	domain := "serviceaccount." + tenant.Identifier
	if sa.BuiltinType != "" {
		domain = sa.BuiltinType + "." + domain
	}
	return name + "@" + domain
}
