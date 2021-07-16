package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

func MergeSchema(iamSchema *memdb.DBSchema) (*memdb.DBSchema, error) {
	included := []*memdb.DBSchema{
		ServerSchema(),
	}

	for _, s := range included {
		for name, table := range s.Tables {
			if _, ok := iamSchema.Tables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			iamSchema.Tables[name] = table
		}
	}

	err := iamSchema.Validate()
	if err != nil {
		return nil, err
	}
	return iamSchema, nil
}
