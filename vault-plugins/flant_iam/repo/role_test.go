package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_RoleDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: RoleSchema()}).Validate(); err != nil {
		t.Fatalf("role schema is invalid: %v", err)
	}
}
