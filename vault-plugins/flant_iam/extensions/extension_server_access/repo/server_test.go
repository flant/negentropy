package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_ServerDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: ServerSchema()}).Validate(); err != nil {
		t.Fatalf("server schema is invalid: %v", err)
	}
}
