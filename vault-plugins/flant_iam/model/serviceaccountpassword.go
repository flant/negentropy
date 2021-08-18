package model

import (
	"time"
)

const ServiceAccountPasswordType = "service_account_password" // also, memdb schema name

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

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (s *ServiceAccountPassword) IsDeleted() bool {
	return s.ArchivingTimestamp != 0
}

func (s *ServiceAccountPassword) ObjType() string {
	return ServiceAccountPasswordType
}

func (s *ServiceAccountPassword) ObjId() string {
	return s.UUID
}
