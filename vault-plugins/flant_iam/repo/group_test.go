package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_GroupDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: GroupSchema()}).Validate(); err != nil {
		t.Fatalf("group schema is invalid: %v", err)
	}
}
