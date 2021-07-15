package model

import (
	"testing"

)

func Test_UserDbSchema(t *testing.T) {
	schema := UserSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("user schema is invalid: %v", err)
	}
}
