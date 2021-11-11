package model

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"
)

func GetSchema(onlyJwks bool) (map[string]*hcmemdb.TableSchema, error) {
	result := JWKSSchema()
	if onlyJwks {
		return result, nil
	}

	others := []map[string]*hcmemdb.TableSchema{
		ConfigSchema(),
		StateSchema(),
	}

	for _, tables := range others {
		for name, table := range tables {
			if _, ok := result[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			result[name] = table
		}
	}
	return result, nil
}
