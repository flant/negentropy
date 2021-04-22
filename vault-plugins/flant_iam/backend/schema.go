package backend

import "github.com/hashicorp/vault/sdk/framework"

type Schema interface {
	Fields() map[string]*framework.FieldSchema
}

type TenantSchema struct{}

func (s TenantSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		"name": {Type: framework.TypeString, Description: "Tenant name"},
	}
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
