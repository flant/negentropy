package model

import (
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_TenantMarshalling(t *testing.T) {
	ten := &Tenant{
		UUID:       uuid.New(),
		Identifier: "somefun",
	}

	json, err := ten.Marshal(false)
	if err != nil {
		t.Fatalf("cannot marshal tenant with sensitive data: %v", err)
	}

	ten2 := &Tenant{}
	err = ten2.Unmarshal(json)
	if err != nil {
		t.Fatalf("cannot unmarshal tenant back: %v", err)
	}

	if !reflect.DeepEqual(ten, ten2) {
		t.Fatalf("tenant changed during marshalling/unmarshalling: was=%v, became=%v", ten, ten2)
	}
}

func Test_TenantDbSchema(t *testing.T) {
	schema := TenantSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("tenant schema is invalid: %v", err)
	}
}
