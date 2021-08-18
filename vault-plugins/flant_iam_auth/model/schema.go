package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
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

		iam_model.UserSchema(),
		iam_model.TenantSchema(),
		iam_model.ProjectSchema(),
		iam_model.ServiceAccountSchema(),
		iam_model.FeatureFlagSchema(),
		iam_model.GroupSchema(),
		iam_model.RoleSchema(),
		iam_model.RoleBindingSchema(),
		iam_model.RoleBindingApprovalSchema(),
		iam_model.MultipassSchema(),
		iam_model.ServiceAccountPasswordSchema(),
		iam_model.IdentitySharingSchema(),
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
