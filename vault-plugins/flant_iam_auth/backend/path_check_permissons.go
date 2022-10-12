package backend

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	authz2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func pathCheckPermissions(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `check_permissions$`,
		Fields: map[string]*framework.FieldSchema{
			"method": {
				Type:        framework.TypeLowerCaseString,
				Description: "The auth method.",
				Required:    true,
			},
			"roles": {
				Type:        framework.TypeSlice,
				Description: "Requested roles",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			// needs Create or Update for passing data, otherwise vault cuts any request body
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.сheckPermissionsHandler,
				Summary:  "Check requested roles for availability",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.сheckPermissionsHandler,
				Summary:  "Check requested roles for availability",
			},
		},
	}
}

func (b *flantIamAuthBackend) сheckPermissionsHandler(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	logger := b.NamedLogger("CheckPermissions")

	// auth_method
	methodName := d.Get("method").(string)
	if methodName == "" {
		return backentutils.ResponseErr(req, fmt.Errorf("%w:missing method", consts.ErrInvalidArg))
	}

	// collect role_claims
	roleClaims, err := getRoleClaims(d)
	if err != nil {
		return nil, fmt.Errorf("%w:parsing roles:%s", consts.ErrInvalidArg, err.Error())
	}

	txn := b.storage.Txn(false)
	defer txn.Abort()

	entityIDOwner, err := b.entityIDResolver.RevealEntityIDOwner(req.EntityID, txn, req.Storage)
	if errors.Is(err, consts.ErrNotFound) {
		return logical.RespondWithStatusCode(nil, req, http.StatusNotFound) //nolint:errCheck
	}
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	// collect subject
	subject, err := buildSubject(*entityIDOwner)
	if err != nil {
		logger.Error(err.Error())
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	authorizator := authz2.NewAutorizator(txn, b.accessVaultClientProvider, b.accessorGetter, logger)
	return logical.RespondWithStatusCode(&logical.Response{
		Data: map[string]interface{}{
			"permissions": authorizator.CheckPermissions(methodName, *subject, roleClaims),
		},
	}, req, http.StatusOK)
}

func buildSubject(entityIDOwner authn.EntityIDOwner) (*model2.Subject, error) {
	switch entityIDOwner.OwnerType {
	case iam.UserType:
		{
			user, ok := entityIDOwner.Owner.(*iam.User)
			if !ok {
				return nil, fmt.Errorf("can't cast, need *model.User, got: %T", entityIDOwner.Owner)
			}
			return &model2.Subject{
				Type:       iam.UserType,
				UUID:       user.UUID,
				TenantUUID: user.TenantUUID,
			}, nil
		}

	case iam.ServiceAccountType:
		{
			sa, ok := entityIDOwner.Owner.(*iam.ServiceAccount)
			if !ok {
				return nil, fmt.Errorf("can't cast, need *model.ServiceAccount, got: %T", entityIDOwner.Owner)
			}
			return &model2.Subject{
				Type:       iam.ServiceAccountType,
				UUID:       sa.UUID,
				TenantUUID: sa.TenantUUID,
			}, nil
		}
	default:
		return nil, fmt.Errorf("wrong entityIDOwner type: %s",
			entityIDOwner.OwnerType)
	}
}
