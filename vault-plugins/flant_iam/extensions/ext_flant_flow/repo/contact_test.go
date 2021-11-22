package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_ContactDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: ContactSchema()}).Validate(); err != nil {
		t.Fatalf("user schema is invalid: %v", err)
	}
}
