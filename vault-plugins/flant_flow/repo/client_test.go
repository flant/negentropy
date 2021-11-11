package repo

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_ClientMarshalling(t *testing.T) {
	ten := &model.Client{
		Tenant: iam_model.Tenant{
			UUID:       uuid.New(),
			Identifier: "somefun",
		},
	}

	raw, err := json.Marshal(ten)
	if err != nil {
		t.Fatalf("cannot marshal tenant with sensitive data: %v", err)
	}

	ten2 := &model.Client{}
	err = json.Unmarshal(raw, &ten2)
	if err != nil {
		t.Fatalf("cannot unmarshal tenant back: %v", err)
	}

	if !reflect.DeepEqual(ten, ten2) {
		t.Fatalf("tenant changed during marshalling/unmarshalling: was=%v, became=%v", ten, ten2)
	}
}

func Test_ClientDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: ClientSchema()}).Validate(); err != nil {
		t.Fatalf("tenant schema is invalid: %v", err)
	}
}
