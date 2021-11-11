package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_MultipassGenerationNumberDbSchema(t *testing.T) {
	if err := (&memdb.DBSchema{Tables: MultipassGenerationNumberSchema()}).Validate(); err != nil {
		t.Fatalf("token generation number schema is invalid: %v", err)
	}
}
