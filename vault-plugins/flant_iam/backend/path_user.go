package backend

//
//import (
//	"github.com/hashicorp/vault/sdk/framework"
//	"github.com/hashicorp/vault/sdk/logical"
//)
//
//type UserSchema struct {
//	uuidGenerator
//}
//
//func (s *UserSchema) ParseEntry(entry *logical.StorageEntry) (Data, error) {
//	panic("implement me")
//}
//
//func (s *UserSchema) ParseData(data *framework.FieldData) (Data, error) {
//	panic("implement me")
//}
//
//func (s *UserSchema) Type() string {
//	return "user"
//}
//
//func (s *UserSchema) SyncTopics() []Topic {
//	panic("implement me")
//}
//
//func (s *UserSchema) Fields() map[string]*framework.FieldSchema {
//	return map[string]*framework.FieldSchema{
//		// TODO unique within tenant
//		"login": {Type: framework.TypeString, Description: "User login"},
//
//		// TODO unique globally or per tenant?
//		"email": {Type: framework.TypeString, Description: "User email"},
//
//		"mobile_phone": {Type: framework.TypeString, Description: "User mobile_phone"},
//
//		"first_name":   {Type: framework.TypeString, Description: "User first_name"},
//		"last_name":    {Type: framework.TypeString, Description: "User last_name"},
//		"display_name": {Type: framework.TypeString, Description: "User display_name"},
//
//		"additional_emails": {Type: framework.TypeCommaStringSlice, Description: "User additional_emails"},
//		"additional_phones": {Type: framework.TypeCommaStringSlice, Description: "User additional_phones"},
//	}
//}
//
//func (s *UserSchema) Validate(data *framework.FieldData) error {
//	return nil // TODO
//}
