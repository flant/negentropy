package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const UserType = "user" // also, memdb schema name

type User struct {
	memdb.ArchivableImpl

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
}

func (u *User) ObjType() string {
	return UserType
}

func (u *User) ObjId() string {
	return u.UUID
}
