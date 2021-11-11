package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_RoleBindingDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: RoleBindingSchema()}).Validate(); err != nil {
		t.Fatalf("role binding schema is invalid: %v", err)
	}
}
