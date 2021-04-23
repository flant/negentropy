package backend

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Group struct {
	Identifier string `json:"identifier"`
}

type GroupSchema struct {
	uuidGenerator
}

func (s *GroupSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
	panic("implement me")
}

func (s *GroupSchema) ParseData(data *framework.FieldData) (Data, error) {
	panic("implement me")
}

func (s *GroupSchema) Type() string {
	return "group"
}

func (s *GroupSchema) SyncTopics() []Topic {
	return []Topic{
		Vault,
		Metadata,
	}
}

func (s *GroupSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s *GroupSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}
