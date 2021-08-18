package repo

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func Test_ServerDbSchema(t *testing.T) {
	schema := model.TenantSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("server schema is invalid: %v", err)
	}
}
