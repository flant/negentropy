package repo

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	jwt_model "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	ID       = "id" // required index by all tables
	ByName   = "name"
	ByUserID = "user_id"
)

func mergeTables() (map[string]*hcmemdb.TableSchema, error) {
	others := []map[string]*hcmemdb.TableSchema{
		EntitySchema(),
		EntityAliasSchema(),
		AuthSourceSchema(),
		AuthMethodSchema(),
		JWTIssueTypeSchema(),
		MultipassGenerationNumberSchema(),
	}

	allTables := map[string]*memdb.TableSchema{}

	for _, tables := range others {
		for name, table := range tables {
			if _, ok := allTables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			allTables[name] = table
		}
	}
	return allTables, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	tables, err := mergeTables()
	if err != nil {
		return nil, err
	}

	jwtSchema, err := jwt_model.GetSchema(false)
	if err != nil {
		return nil, err
	}

	schema, err := memdb.MergeDBSchemas(
		&memdb.DBSchema{
			Tables: tables,
		},
		memdb.DropRelations(iam_repo.TenantSchema()),
		memdb.DropRelations(iam_repo.ProjectSchema()),
		memdb.DropRelations(iam_repo.GroupSchema()),
		memdb.DropRelations(iam_repo.UserSchema()),
		memdb.DropRelations(iam_repo.FeatureFlagSchema()),
		memdb.DropRelations(iam_repo.ServiceAccountSchema()),
		memdb.DropRelations(iam_repo.RoleSchema()),
		memdb.DropRelations(iam_repo.RoleBindingSchema()),
		memdb.DropRelations(iam_repo.RoleBindingApprovalSchema()),
		memdb.DropRelations(iam_repo.MultipassSchema()),
		memdb.DropRelations(iam_repo.ServiceAccountPasswordSchema()),
		memdb.DropRelations(iam_repo.IdentitySharingSchema()),
		memdb.DropRelations(ext_repo.ServerSchema()),

		jwtSchema,
	)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func NewResourceVersion() string {
	return uuid.New()
}
