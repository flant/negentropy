package model

import (
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const MultipassType = "multipass" // also, memdb schema name

type MultipassOwnerType = string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

type Multipass struct {
	memdb.ArchivableImpl

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

	Origin consts.ObjectOrigin `json:"origin"`

	Extensions map[consts.ObjectOrigin]*Extension `json:"-"`
}

func (m *Multipass) ObjType() string {
	return MultipassType
}

func (m *Multipass) ObjId() string {
	return m.UUID
}
