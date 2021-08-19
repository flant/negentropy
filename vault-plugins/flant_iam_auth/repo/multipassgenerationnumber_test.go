package repo

import (
	"testing"
)

func Test_MultipassGenerationNumberDbSchema(t *testing.T) {
	schema := MultipassGenerationNumberSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("token generation number schema is invalid: %v", err)
	}
}
