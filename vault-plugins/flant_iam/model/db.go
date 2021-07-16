package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func mergeSchema() (*memdb.DBSchema, error) {
	jwtSchema, err := model.GetSchema(false)
	if err != nil {
		return nil, err
	}

	included := []*memdb.DBSchema{
		TenantSchema(),
		UserSchema(),
		ReplicaSchema(),
		ProjectSchema(),
		FeatureFlagSchema(),
		ServiceAccountSchema(),
		GroupSchema(),
		RoleSchema(),
		RoleBindingSchema(),
		RoleBindingApprovalSchema(),
		MultipassSchema(),
		ServiceAccountPasswordSchema(),
		IdentitySharingSchema(),

		jwtSchema,
	}

	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{},
	}

	for _, s := range included {
		for name, table := range s.Tables {
			if _, ok := schema.Tables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			schema.Tables[name] = table
		}
	}

	err = schema.Validate()
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	schema, err := mergeSchema()
	if err != nil {
		return nil, err
	}
	err = schema.Validate()
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func NewResourceVersion() string {
	return uuid.New()
}
