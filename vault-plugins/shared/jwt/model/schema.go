package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

func GetSchema(onlyJwks bool) (*memdb.DBSchema, error) {
	schema := JWKSSchema()
	if onlyJwks {
		return schema, nil
	}

	others := []*memdb.DBSchema{
		ConfigSchema(),
		StateSchema(),
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
