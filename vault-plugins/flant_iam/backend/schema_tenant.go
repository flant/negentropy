package backend

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type TenantSchema struct {
	uuidGenerator
}

func (s *TenantSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
	panic("implement me")
}

func (s *TenantSchema) ParseData(data *framework.FieldData) (Data, error) {
	panic("implement me")
}

func (s *TenantSchema) Type() string {
	return "tenant"
}

func (s *TenantSchema) SyncTopics() []Topic {
	panic("implement me")
}

func (s *TenantSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"identifier": {
			Type:        framework.TypeNameString,
			Description: "Identifier for humans and machines",
			Required:    true, // seems to work for doc, not validation
		},
	}
}

func (s *TenantSchema) Validate(data *framework.FieldData) error {
	name, ok := data.GetOk("identifier")
	if !ok || len(name.(string)) == 0 {
		return fmt.Errorf("tenant identifier must not be empty")
	}
	return nil
}
