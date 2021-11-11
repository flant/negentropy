package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_ServiceAccountDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: ServiceAccountSchema()}).Validate(); err != nil {
		t.Fatalf("service account schema is invalid: %v", err)
	}
}
