package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

const (
	ID = "id" // required index by all tables
)

func mergeSchema() (*memdb.DBSchema, error) {
	schema := TenantSchema()
	others := []*memdb.DBSchema{
		UserSchema(),
		ReplicaSchema(),
		ProjectSchema(),
		FeatureFlagSchema(),
	}

	for _, o := range others {
		for name, table := range o.Tables {
			if _, ok := schema.Tables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			schema.Tables[name] = table
		}
	}
	return schema, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	return mergeSchema()
}

func NewResourceVersion() string {
	return uuid.New()
}
