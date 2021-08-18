package model

import (
	"time"
)

const MultipassType = "multipass" // also, memdb schema name

type MultipassOwnerType = string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

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

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (m *Multipass) IsDeleted() bool {
	return m.ArchivingTimestamp != 0
}

func (m *Multipass) ObjType() string {
	return MultipassType
}

func (m *Multipass) ObjId() string {
	return m.UUID
}
