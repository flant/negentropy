package repo

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func mergeSchema() (*memdb.DBSchema, error) {
	included := []*memdb.DBSchema{
		ClientSchema(),
		TeamSchema(),
		TeammateSchema(),
		ProjectSchema(),
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

	err := schema.Validate()
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
