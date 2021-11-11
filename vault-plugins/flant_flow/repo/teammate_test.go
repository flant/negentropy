package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_TeamMateDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: TeammateSchema()}).Validate(); err != nil {
		t.Fatalf("teammate schema is invalid: %v", err)
	}
}
