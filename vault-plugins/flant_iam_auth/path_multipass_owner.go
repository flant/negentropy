package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extension_server_access/model"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
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

// revealEntityIDOwner returns type and info about token owner
// it can be iam.User, or iam.ServiceAccount
func (b *flantIamAuthBackend) revealEntityIDOwner(ctx context.Context,
	req *logical.Request) (string, interface{}, error) {
	logger := b.NamedLogger("revealEntityIDOwner")
	logger.Debug(fmt.Sprintf("EntityID=%s", req.EntityID))
	vaultClient, err := b.accessVaultController.APIClient()
	if err != nil {
		return "", nil, fmt.Errorf("internal error accessing vault client: %w", err)
	}

	entityApi := api.NewIdentityAPI(vaultClient, logger.Named("LoginIdentityApi")).EntityApi()
	ent, err := entityApi.GetByID(req.EntityID)
	if err != nil {
		return "", nil, fmt.Errorf("finding vault entity by id: %w", err)
	}

	name, ok := ent["name"]
	if !ok {
		return "", nil, fmt.Errorf("field 'name' in vault entity is ommited")
	}

	nameStr, ok := name.(string)
	if !ok {
		return "", nil, fmt.Errorf("field 'name' should be string")
	}
	logger.Debug(fmt.Sprintf("catch name of vault entity: %s", nameStr))

	txn := b.storage.Txn(false)
	defer txn.Abort()

	iamEntity, err := repo.NewEntityRepo(txn).GetByName(nameStr)
	if err != nil {
		return "", nil, fmt.Errorf("finding iam_entity by name:%w", err)
	}
	logger.Debug(fmt.Sprintf("catch multipass owner UUID: %s, try to find user", iamEntity.UserId))

	user, err := iam_repo.NewUserRepository(txn).GetByID(iamEntity.UserId)
	if err != nil && !errors.Is(err, iam.ErrNotFound) {
		return "", nil, fmt.Errorf("finding user by id:%w", err)
	}
	if err == nil {
		logger.Debug(fmt.Sprintf("found user UUID: %s", user.UUID))
		return iam.UserType, user, nil
	} else {
		logger.Debug("Not found user, try to find service_account")
		sa, err := iam_repo.NewServiceAccountRepository(txn).GetByID(iamEntity.UUID)
		if err != nil && !errors.Is(err, iam.ErrNotFound) {
			return "", nil, fmt.Errorf("finding service_account by id:%w", err)
		}
		if errors.Is(err, iam.ErrNotFound) {
			logger.Debug("Not found neither user nor service_account")
			return "", nil, err
		}
		logger.Debug(fmt.Sprintf("found service_account UUID: %s", sa.UUID))
		return iam.ServiceAccountType, sa, nil
	}
}

func (b *flantIamAuthBackend) multipassOwner(ctx context.Context, req *logical.Request,
	d *framework.FieldData) (*logical.Response, error) {
	logger := b.NamedLogger("multipassOwner")
	subjectType, subject, err := b.revealEntityIDOwner(ctx, req)
	if errors.Is(err, iam.ErrNotFound) {
		return logical.RespondWithStatusCode(nil, req, http.StatusNotFound) //nolint:errCheck
	}
	if err != nil {
		return responseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	switch subjectType {
	case iam.UserType:
		{
			user, ok := subject.(*iam.User)
			if !ok {
				err := fmt.Errorf("can't cast, need *model.User, got: %T", subject)
				logger.Debug(err.Error())
				return responseErrMessage(req, err.Error(), http.StatusInternalServerError)
			}
			return logical.RespondWithStatusCode(&logical.Response{
				Data: map[string]interface{}{
					"user": ext.User{
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

	case iam.ServiceAccountType:
		{
			sa, ok := subject.(*iam.ServiceAccount)
			if !ok {
				err := fmt.Errorf("can't cast, need *model.ServiceAccount, got: %T", subject)
				logger.Debug(err.Error())
				return responseErrMessage(req, err.Error(), http.StatusInternalServerError)
			}
			return logical.RespondWithStatusCode(&logical.Response{
				Data: map[string]interface{}{
					"service_account": ext.ServiceAccount{
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
	}
	msg := fmt.Sprintf("unexpected subjectType: `%s`", subjectType)
	logger.Debug(msg)
	return responseErrMessage(req, err.Error(), http.StatusInternalServerError)
}
