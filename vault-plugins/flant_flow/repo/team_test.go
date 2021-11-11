package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_TeamDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: TeamSchema()}).Validate(); err != nil {
		t.Fatalf("team schema is invalid: %v", err)
	}
}
