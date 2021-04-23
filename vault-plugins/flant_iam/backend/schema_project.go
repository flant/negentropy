package backend

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type ProjectSchema struct {
	uuidGenerator
}

func (s *ProjectSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
	panic("implement me")
}

func (s *ProjectSchema) ParseData(data *framework.FieldData) (Data, error) {
	panic("implement me")
}

func (s *ProjectSchema) Type() string {
	return "project"
}

func (s *ProjectSchema) SyncTopics() []Topic {
	panic("implement me")
}

func (s *ProjectSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s *ProjectSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}
