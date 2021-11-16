package model

import (
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const ServiceAccountPasswordType = "service_account_password" // also, memdb schema name

type ServiceAccountPassword struct {
	memdb.ArchivableImpl

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

func (s *ServiceAccountPassword) ObjType() string {
	return ServiceAccountPasswordType
}

func (s *ServiceAccountPassword) ObjId() string {
	return s.UUID
}
