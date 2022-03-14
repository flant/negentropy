package repo

import (
	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
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

func GetSchema() (*memdb.DBSchema, error) {
	jwtSchema, err := jwt_model.GetSchema(false)
	if err != nil {
		return nil, err
	}

	schema, err := memdb.MergeDBSchemasAndValidate(
		// TODO add relations into next schemas
		EntitySchema(),
		EntityAliasSchema(),
		AuthSourceSchema(),
		AuthMethodSchema(),
		JWTIssueTypeSchema(),
		MultipassGenerationNumberSchema(),
		PolicySchema(),

		// copy of data from iam, so no needs to checks
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
