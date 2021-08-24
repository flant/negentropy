package jwtauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
)

func pathMultipassOwnner(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `multipass_owner$`,
		Fields: map[string]*framework.FieldSchema{
			"multipass": {
				Type:        framework.TypeString,
				Description: "multipass jwt",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.multipassOwner,
				Summary:  pathLoginHelpSyn,
			},
		},

		HelpSynopsis: "Provide info about owner of multipass",
	}
}

func (b *flantIamAuthBackend) multipassOwner(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	logger := b.NamedLogger("multipassOwner")
	logger.Debug(fmt.Sprintf("multipassOwner : EntityID=%s", req.EntityID))
	vaultClient, err := b.accessVaultController.APIClient()
	if err != nil {
		return responseErr(req, fmt.Errorf("internal error accessing vault client: %w", err))
	}

	entityApi := api.NewIdentityAPI(vaultClient, logger.Named("LoginIdentityApi")).EntityApi()
	ent, err := entityApi.GetByID(req.EntityID)
	if err != nil {
		return responseErr(req, fmt.Errorf("finding vault entity by id: %w", err))
	}

	name, ok := ent["name"]
	if !ok {
		return responseErr(req, fmt.Errorf("field 'name' in vault entity is ommited"))
	}

	nameStr, ok := name.(string)
	if !ok {
		return responseErr(req, fmt.Errorf("field 'name' should be string"))
	}
	logger.Debug(fmt.Sprintf("catch name of vault entity: %s", nameStr))

	tnx := b.storage.Txn(false)
	defer tnx.Abort()

	iamEntity, err := repo.NewEntityRepo(tnx).GetByName(nameStr)
	if err != nil {
		return responseErr(req, fmt.Errorf("finding iam_entity by name::%w", err))
	}
	logger.Debug(fmt.Sprintf("catch multipass owner UUID: %s, try to find user", iamEntity.UserId))

	user, err := iam_repo.NewUserRepository(tnx).GetByID(iamEntity.UserId)
	if err != nil {
		logger.Debug(fmt.Sprintf("err: %s, try to find service_account", err))
		sa, err := iam_repo.NewServiceAccountRepository(tnx).GetByID(iamEntity.UUID)
		if err != nil {
			logger.Debug(fmt.Sprintf("err: %s, not found", err))
			logical.RespondWithStatusCode(nil, req, http.StatusNotFound) //nolint:errCheck
		}
		logger.Debug(fmt.Sprintf("found service_account UUID: %s", sa.UUID))
		return logical.RespondWithStatusCode(&logical.Response{
			Data: map[string]interface{}{
				"service_account": model.ServiceAccount{
					UUID:           sa.UUID,
					TenantUUID:     sa.TenantUUID,
					BuiltinType:    sa.BuiltinType,
					Identifier:     sa.Identifier,
					FullIdentifier: sa.FullIdentifier,
					CIDRs:          sa.CIDRs,
					Origin:         sa.TenantUUID,
				},
			},
		}, req, http.StatusOK)
	}
	logger.Debug(fmt.Sprintf("found user UUID: %s", user.UUID))
	return logical.RespondWithStatusCode(&logical.Response{
		Data: map[string]interface{}{
			"user": model.User{
				UUID:             user.UUID,
				TenantUUID:       user.TenantUUID,
				Origin:           user.TenantUUID,
				Identifier:       user.Identifier,
				FullIdentifier:   user.FullIdentifier,
				FirstName:        user.FirstName,
				LastName:         user.LastName,
				DisplayName:      user.DisplayName,
				Email:            user.Email,
				AdditionalEmails: user.AdditionalEmails,
				MobilePhone:      user.MobilePhone,
				AdditionalPhones: user.AdditionalPhones,
			},
		},
	}, req, http.StatusOK)
}
