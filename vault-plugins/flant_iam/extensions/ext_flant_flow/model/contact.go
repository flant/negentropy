package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const ContactType = "contact" // also, memdb schema name

type Contact struct {
	iam_model.User

	Credentials map[iam_model.ProjectUUID]ContactRole `json:"credentials"`
}

func (c *Contact) IsDeleted() bool {
	return c.Timestamp != 0
}

func (c *Contact) ObjType() string {
	return ContactType
}

func (c *Contact) ObjId() string {
	return c.UUID
}
