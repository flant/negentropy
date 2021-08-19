package repo

import (
	"testing"
)

func Test_ServerDbSchema(t *testing.T) {
	schema := ServerSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("server schema is invalid: %v", err)
	}
}
