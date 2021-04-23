package backend

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type RoleSchema struct {
	uuidGenerator
}

func (s *RoleSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
	panic("implement me")
}

func (s *RoleSchema) ParseData(data *framework.FieldData) (Data, error) {
	panic("implement me")
}

func (s *RoleSchema) Type() string {
	return "role"
}

func (s *RoleSchema) SyncTopics() []Topic {
	panic("implement me")
}

func (s *RoleSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s *RoleSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}
