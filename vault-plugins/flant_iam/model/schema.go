package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

const (
	ID = "id" // required index by all tables
)

func mergeSchema() (*memdb.DBSchema, error) {
	schema := TenantSchema()
	others := []*memdb.DBSchema{
		UserSchema(),
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

func NewDB() (*memdb.MemDB, error) {
	schema, err := mergeSchema()
	if err != nil {
		return nil, err
	}
	return memdb.NewMemDB(schema)
}
