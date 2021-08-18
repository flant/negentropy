package repo

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func Test_FeatureFlagMarshalling(t *testing.T) {
	ten := &model.FeatureFlag{
		Name: "somefun",
	}

	raw, err := json.Marshal(ten)
	if err != nil {
		t.Fatalf("cannot marshal tenant with sensitive data: %v", err)
	}

	ten2 := &model.FeatureFlag{}
	err = json.Unmarshal(raw, ten2)
	if err != nil {
		t.Fatalf("cannot unmarshal tenant back: %v", err)
	}

	if !reflect.DeepEqual(ten, ten2) {
		t.Fatalf("tenant changed during marshalling/unmarshalling: was=%v, became=%v", ten, ten2)
	}
}

func Test_FeatureFlagDbSchema(t *testing.T) {
	schema := FeatureFlagSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("tenant schema is invalid: %v", err)
	}
}
