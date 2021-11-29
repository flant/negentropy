package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const ContactType = "contact" // also, memdb schema name

type FullContact struct {
	iam_model.User

	Credentials map[iam_model.ProjectUUID]ContactRole `json:"credentials"`
}

type Contact struct {
	memdb.ArchiveMark
	UserUUID    iam_model.UserUUID                    `json:"user_uuid"`
	TenantUUID  iam_model.TenantUUID                  `json:"tenant_uuid"`
	Credentials map[iam_model.ProjectUUID]ContactRole `json:"credentials"`
	Version     string                                `json:"resource_version"`
}

func (c *Contact) ObjType() string {
	return ContactType
}

func (c *Contact) ObjId() string {
	return c.UserUUID
}

func (f *FullContact) GetContact() *Contact {
	if f == nil {
		return nil
	}
	return &Contact{
		ArchiveMark: f.ArchiveMark,
		UserUUID:    f.UUID,
		TenantUUID:  f.TenantUUID,
		Credentials: f.Credentials,
		Version:     f.Version,
	}
}
