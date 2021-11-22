package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const ClientType = "client" // also, memdb schema name

type Client struct {
	iam_model.Tenant
}

func (t *Client) IsDeleted() bool {
	return t.ArchivingTimestamp != 0
}

func (t *Client) ObjType() string {
	return ClientType
}

func (t *Client) ObjId() string {
	return t.UUID
}
