package model

const UserType = "user" // also, memdb schema name

type User struct {
	UUID       UserUUID   `json:"uuid"` // PK
	TenantUUID TenantUUID `json:"tenant_uuid"`
	Version    string     `json:"resource_version"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`

	Identifier     string `json:"identifier"`
	FullIdentifier string `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`

	Email            string   `json:"email"`
	AdditionalEmails []string `json:"additional_emails"`

	MobilePhone      string   `json:"mobile_phone"`
	AdditionalPhones []string `json:"additional_phones"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (u *User) IsDeleted() bool {
	return u.ArchivingTimestamp != 0
}

func (u *User) ObjType() string {
	return UserType
}

func (u *User) ObjId() string {
	return u.UUID
}
