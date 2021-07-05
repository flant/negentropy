package model

import (
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_ExtensionMarshalling(t *testing.T) {
	ten := &Extension{
		UUID: uuid.New(),

		OwnerType: UserType,
		OwnerUUID: uuid.New(),

		Attributes:          map[string]interface{}{"key": "value"},
		SensitiveAttributes: map[string]interface{}{"sensitive_key": "sensitive_value"},
	}

	json, err := ten.Marshal(false)
	if err != nil {
		t.Fatalf("cannot marshal extension with sensitive data: %v", err)
	}

	ten2 := &Extension{}
	err = ten2.Unmarshal(json)
	if err != nil {
		t.Fatalf("cannot unmarshal extension back: %v", err)
	}

	if !reflect.DeepEqual(ten, ten2) {
		t.Fatalf("extension changed during marshalling/unmarshalling: was=%v, became=%v", ten, ten2)
	}
}
