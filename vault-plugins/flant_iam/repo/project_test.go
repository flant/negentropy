package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_ProjectDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: ProjectSchema()}).Validate(); err != nil {
		t.Fatalf("Project schema is invalid: %v", err)
	}
}
