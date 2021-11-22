package model

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func GetSchema(onlyJwks bool) (*memdb.DBSchema, error) {
	allTables := JWKSTables()
	if onlyJwks {
		return &memdb.DBSchema{Tables: allTables}, nil
	}

	otherTables := []map[string]*hcmemdb.TableSchema{
		ConfigTables(),
		StateTables(),
	}

	for _, tables := range otherTables {
		for name, table := range tables {
			if _, ok := allTables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			allTables[name] = table
		}
	}
	return &memdb.DBSchema{Tables: allTables}, nil
}
