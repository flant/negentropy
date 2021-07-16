package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
)

func ServiceAccountPasswordSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServiceAccountPasswordType: {
				Name: ServiceAccountPasswordType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					OwnerForeignPK: {
						Name: OwnerForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "OwnerUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

//go:generate go run gen_repository.go -type ServiceAccountPassword -parentType Owner
type ServiceAccountPassword struct {
	UUID       ServiceAccountPasswordUUID `json:"uuid"` // PK
	TenantUUID TenantUUID                 `json:"tenant_uuid"`
	OwnerUUID  OwnerUUID                  `json:"owner_uuid"`

	Description string `json:"description"`

	CIDRs []string   `json:"allowed_cidrs"`
	Roles []RoleName `json:"allowed_roles" `

	TTL       time.Duration `json:"ttl"`
	ValidTill int64         `json:"valid_till"` // calculates from TTL on creation

	Secret string `json:"secret,omitempty" sensitive:""` // generates on creation
}
