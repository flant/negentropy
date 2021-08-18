package model

const IdentitySharingType = "identity_sharing" // also, memdb schema name

type IdentitySharing struct {
	UUID                  IdentitySharingUUID `json:"uuid"` // PK
	SourceTenantUUID      TenantUUID          `json:"source_tenant_uuid"`
	DestinationTenantUUID TenantUUID          `json:"destination_tenant_uuid"`

	Version string `json:"resource_version"`

	// Groups which to share with target tenant
	Groups []GroupUUID `json:"groups"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (i *IdentitySharing) IsDeleted() bool {
	return i.ArchivingTimestamp != 0
}

func (i *IdentitySharing) ObjType() string {
	return IdentitySharingType
}

func (i *IdentitySharing) ObjId() string {
	return i.UUID
}
