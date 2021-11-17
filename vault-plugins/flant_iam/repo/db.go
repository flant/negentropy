package repo

import (
	"fmt"

	jwt "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func mergeTables() (map[string]*memdb.TableSchema, error) {
	jwtSchema, err := jwt.GetSchema(false)
	if err != nil {
		return nil, err
	}

	included := []map[string]*memdb.TableSchema{
		ReplicaSchema(),
		//
		jwtSchema,
	}

	tables := map[string]*memdb.TableSchema{}

	for _, partialTables := range included {
		for name, table := range partialTables {
			if _, ok := tables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			tables[name] = table
		}
	}

	if err != nil {
		return nil, err
	}
	return tables, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	tables, err := mergeTables()
	if err != nil {
		return nil, err
	}
	schema, err := memdb.MergeDBSchemas(
		&memdb.DBSchema{Tables: tables},
		TenantSchema(),
		ProjectSchema(),
		GroupSchema(),
		UserSchema(),
		FeatureFlagSchema(),
		ServiceAccountSchema(),
		RoleSchema(),
		RoleBindingSchema(),
		RoleBindingApprovalSchema(),
		MultipassSchema(),
		ServiceAccountPasswordSchema(),
		IdentitySharingSchema(),
	)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func NewResourceVersion() string {
	return uuid.New()
}
