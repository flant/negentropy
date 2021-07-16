package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	ServiceAccountType              = "service_account" // also, memdb schema name
	fullIdentifierIndex             = "full_identifier"
	TenantUUIDServiceAccountIdIndex = "tenant_uuid_service_account_id"
)

type ServiceAccountObjectType string

func ServiceAccountSchema() *memdb.DBSchema {
	var tenantUUIDServiceAccountIdIndexer []memdb.Indexer

	tenantUUIDIndexer := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, groupIdIndexer)

	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServiceAccountType: {
				Name: ServiceAccountType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
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
					fullIdentifierIndex: {
						Name: fullIdentifierIndex,
						Indexer: &memdb.StringFieldIndex{
							Field:     "FullIdentifier",
							Lowercase: true,
						},
					},
					TenantUUIDServiceAccountIdIndex: {
						Name:    TenantUUIDServiceAccountIdIndex,
						Indexer: &memdb.CompoundIndex{Indexes: tenantUUIDServiceAccountIdIndexer},
					},
				},
			},
		},
	}
}

//go:generate go run gen_repository.go -type ServiceAccount -parentType Tenant
type ServiceAccount struct {
	UUID           ServiceAccountUUID `json:"uuid"` // PK
	TenantUUID     TenantUUID         `json:"tenant_uuid"`
	Version        string             `json:"resource_version"`
	BuiltinType    string             `json:"-"`
	Identifier     string             `json:"identifier"`
	FullIdentifier string             `json:"full_identifier"`
	CIDRs          []string           `json:"allowed_cidrs"`
	TokenTTL       time.Duration      `json:"token_ttl"`
	TokenMaxTTL    time.Duration      `json:"token_max_ttl"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}

func (r *ServiceAccountRepository) GetByIdentifier(tenantUUID, identifier string) (*ServiceAccount, error) {
	raw, err := r.db.First(ServiceAccountType, TenantUUIDServiceAccountIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*ServiceAccount), err
}

	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
}

// TODO move to usecases
// generic: <identifier>@serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(saID, tenantID string) string {
	domain := "serviceaccount." + tenantID

	return saID + "@" + domain
}
