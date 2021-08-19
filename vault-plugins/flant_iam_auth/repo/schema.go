package repo

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	jwt_model "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

const (
	ID       = "id" // required index by all tables
	ByName   = "name"
	ByUserID = "user_id"
)

func mergeSchema() (*memdb.DBSchema, error) {
	jwtSchema, err := jwt_model.GetSchema(false)
	if err != nil {
		return nil, err
	}

	schema := EntitySchema()
	others := []*memdb.DBSchema{
		EntityAliasSchema(),
		AuthSourceSchema(),
		AuthMethodSchema(),
		JWTIssueTypeSchema(),
		MultipassGenerationNumberSchema(),

		iam_repo.UserSchema(),
		iam_repo.TenantSchema(),
		iam_repo.ProjectSchema(),
		iam_repo.ServiceAccountSchema(),
		iam_repo.FeatureFlagSchema(),
		iam_repo.GroupSchema(),
		iam_repo.RoleSchema(),
		iam_repo.RoleBindingSchema(),
		iam_repo.RoleBindingApprovalSchema(),
		iam_repo.MultipassSchema(),
		iam_repo.ServiceAccountPasswordSchema(),
		iam_repo.IdentitySharingSchema(),
		ext_repo.ServerSchema(),
		jwtSchema,
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

func NewResourceVersion() string {
	return uuid.New()
}
