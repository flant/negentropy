package model

import (
	"time"
)

const ServiceAccountType = "service_account" // also, memdb schema name

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

	Extensions         map[ObjectOrigin]*Extension `json:"-"`
	ArchivingTimestamp UnixTime                    `json:"archiving_timestamp"`
	ArchivingHash      int64                       `json:"archiving_hash"`
}

func (s *ServiceAccount) IsDeleted() bool {
	return s.ArchivingTimestamp != 0
}

func (s *ServiceAccount) ObjType() string {
	return ServiceAccountType
}

func (s *ServiceAccount) ObjId() string {
	return s.UUID
}
