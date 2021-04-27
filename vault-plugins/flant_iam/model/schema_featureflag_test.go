package model

import (
	"reflect"
	"testing"
)

func Test_FeatureFlagMarshalling(t *testing.T) {
	ten := &FeatureFlag{
		Name: "somefun",
	}

	json, err := ten.Marshal(false)
	if err != nil {
		t.Fatalf("cannot marshal tenant with sensitive data: %v", err)
	}

	ten2 := &FeatureFlag{}
	err = ten2.Unmarshal(json)
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
