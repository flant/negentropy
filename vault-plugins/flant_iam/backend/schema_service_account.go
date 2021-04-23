package backend

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type ServiceAccountSchema struct {
	uuidGenerator
}

func (s *ServiceAccountSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
	panic("implement me")
}

func (s *ServiceAccountSchema) ParseData(data *framework.FieldData) (Data, error) {
	panic("implement me")
}

func (s *ServiceAccountSchema) Type() string {
	return "service_account"
}

func (s *ServiceAccountSchema) SyncTopics() []Topic {
	panic("implement me")
}

func (s *ServiceAccountSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{}
}

func (s *ServiceAccountSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}

