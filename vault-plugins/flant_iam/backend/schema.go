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
		"name": {Type: framework.TypeString, Description: "Tenant name"},
	}
}

func (s TenantSchema) Validate(data *framework.FieldData) error {
	name, ok := data.GetOk("name")
	if !ok || len(name.(string)) == 0 {
		return fmt.Errorf("tenant name must not be empty")
	}
	return nil
}

type UserSchema struct{}

func (s UserSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		"login":             {Type: framework.TypeString, Description: "User login"},
		"first_name":        {Type: framework.TypeString, Description: "User first_name"},
		"last_name":         {Type: framework.TypeString, Description: "User last_name"},
		"display_name":      {Type: framework.TypeString, Description: "User display_name"},
		"email":             {Type: framework.TypeString, Description: "User email"},
		"additional_emails": {Type: framework.TypeCommaStringSlice, Description: "User additional_emails"},
		"mobile_phone":      {Type: framework.TypeString, Description: "User mobile_phone"},
		"additional_phones": {Type: framework.TypeCommaStringSlice, Description: "User additional_phones"},
	}
}

func (s UserSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}
