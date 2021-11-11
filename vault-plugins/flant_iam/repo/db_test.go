package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_MergeSchema(t *testing.T) {
	schema, err := mergeTables()
	if err != nil {
		t.Fatalf("cannot merge schema: %v", err)
	}
	if err := (&memdb.DBSchema{Tables: schema}).Validate(); err != nil {
		t.Fatalf("merged schema is invalid: %v", err)
	}
}
