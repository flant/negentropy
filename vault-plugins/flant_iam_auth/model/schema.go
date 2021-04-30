package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

const (
	// Index "id" is required by all table.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func mergeSchema() (*memdb.DBSchema, error) {
	schema := PendingLoginSchema()
	others := []*memdb.DBSchema{
		userSchema,
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
