package backend

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
)

type Schema interface {
	Fields() map[string]*framework.FieldSchema
	Validate(*framework.FieldData) error
}

type TenantSchema struct{}

func (s TenantSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"identifier": {
			Type:        framework.TypeNameString,
			Description: "Identifier for humans and machines",
			Required:    true, // seems to work for doc, not validation
		},
	}
}

func (s TenantSchema) Validate(data *framework.FieldData) error {
	name, ok := data.GetOk("identifier")
	if !ok || len(name.(string)) == 0 {
		return fmt.Errorf("tenant identifier must not be empty")
	}
	return nil
}

type UserSchema struct{}

func (s UserSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"login": {Type: framework.TypeString, Description: "User login"},

		// TODO unique within tenant ?
		"email":        {Type: framework.TypeString, Description: "User email"},
		"mobile_phone": {Type: framework.TypeString, Description: "User mobile_phone"},

		"first_name":   {Type: framework.TypeString, Description: "User first_name"},
		"last_name":    {Type: framework.TypeString, Description: "User last_name"},
		"display_name": {Type: framework.TypeString, Description: "User display_name"},

		"additional_emails": {Type: framework.TypeCommaStringSlice, Description: "User additional_emails"},
		"additional_phones": {Type: framework.TypeCommaStringSlice, Description: "User additional_phones"},
	}
}

func (s UserSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}

type ProjectSchema struct{}

func (s ProjectSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s ProjectSchema) Validate(data *framework.FieldData) error {
	return nil
}

type ServiceAccountSchema struct{}

func (s ServiceAccountSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{}
}

func (s ServiceAccountSchema) Validate(data *framework.FieldData) error {
	return nil
}

type GroupSchema struct{}

func (s GroupSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s GroupSchema) Validate(data *framework.FieldData) error {
	return nil
}

type RoleSchema struct{}

func (s RoleSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s RoleSchema) Validate(data *framework.FieldData) error {
	return nil
}
