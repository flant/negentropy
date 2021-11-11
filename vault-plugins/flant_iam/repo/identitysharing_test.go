package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_IdentitySharingDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: IdentitySharingSchema()}).Validate(); err != nil {
		t.Fatalf("identity sharing schema is invalid: %v", err)
	}
}
