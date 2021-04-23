package backend

import (
	"fmt"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/framework"
)

func genUUID() string {
	id, err := uuid.GenerateUUID()
	if err != nil {
		id = genUUID()
	}
	return id
}

type uuidGenerator struct{}

func (g *uuidGenerator) GenerateID() string {
	return genUUID()
}

type Schema interface {
	Fields() map[string]*framework.FieldSchema
	Validate(*framework.FieldData) error
	GenerateID() string
}

type TenantSchema struct {
	uuidGenerator
}

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

type UserSchema struct {
	uuidGenerator
}

func (s UserSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant
		"login": {Type: framework.TypeString, Description: "User login"},

		// TODO unique globally or per tenant?
		"email": {Type: framework.TypeString, Description: "User email"},

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

type ProjectSchema struct {
	uuidGenerator
}

func (s ProjectSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s ProjectSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}

type ServiceAccountSchema struct {
	uuidGenerator
}

func (s ServiceAccountSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{}
}

func (s ServiceAccountSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}

type GroupSchema struct {
	uuidGenerator
}

func (s GroupSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s GroupSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}

type RoleSchema struct {
	uuidGenerator
}

func (s RoleSchema) Fields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		// TODO unique within tenant?
		"identifier": {Type: framework.TypeNameString, Description: "Identifier for humans and machines"},
	}
}

func (s RoleSchema) Validate(data *framework.FieldData) error {
	return nil // TODO
}
