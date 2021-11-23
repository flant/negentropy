package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const IdentitySharingType = "identity_sharing" // also, memdb schema name

type IdentitySharing struct {
	memdb.ArchiveMark

	UUID                  IdentitySharingUUID `json:"uuid"` // PK
	SourceTenantUUID      TenantUUID          `json:"source_tenant_uuid"`
	DestinationTenantUUID TenantUUID          `json:"destination_tenant_uuid"`

	Version string `json:"resource_version"`

	// Groups which to share with target tenant
	Groups []GroupUUID `json:"groups"`
}

func (i *IdentitySharing) ObjType() string {
	return IdentitySharingType
}

func (i *IdentitySharing) ObjId() string {
	return i.UUID
}
