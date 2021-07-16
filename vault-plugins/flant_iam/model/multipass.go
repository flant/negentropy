package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	OwnerForeignPK = "owner_uuid"
)

func MultipassSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			MultipassType: {
				Name: MultipassType,
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

type MultipassOwnerType string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

//go:generate go run gen_repository.go -type Multipass -parentType Owner
type Multipass struct {
	UUID       MultipassUUID      `json:"uuid"` // PK
	TenantUUID TenantUUID         `json:"tenant_uuid"`
	OwnerUUID  OwnerUUID          `json:"owner_uuid"`
	OwnerType  MultipassOwnerType `json:"owner_type"`

	Description string        `json:"description"`
	TTL         time.Duration `json:"ttl"`
	MaxTTL      time.Duration `json:"max_ttl"`
	CIDRs       []string      `json:"allowed_cidrs"`
	Roles       []RoleName    `json:"allowed_roles" `

	ValidTill int64  `json:"valid_till"`
	Salt      string `json:"salt,omitempty" sensitive:""`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}
