package backend

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	authz2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func pathCheckEffectiveRoles(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `check_effective_roles$`,
		Fields: map[string]*framework.FieldSchema{
			"roles": {
				Type:        framework.TypeStringSlice,
				Description: "Requested roles",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			// needs Create or Update for passing data, otherwise vault cuts any request body
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.сheckEffectiveRolesHandler,
				Summary:  "Check requested roles for availability",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.сheckEffectiveRolesHandler,
				Summary:  "Check requested roles for availability",
			},
		},
	}
}

func (b *flantIamAuthBackend) сheckEffectiveRolesHandler(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	logger := b.NamedLogger("CheckEffectiveRolesHandler")

	// collect role_claims
	roles := d.Get("roles").([]string)

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

	authorizator := authz2.NewAutorizator(txn, b.accessVaultProvider, b.accessorGetter, logger)
	effectiveRolesMap, err := authorizator.EffectiveRoleChecker.CheckEffectiveRoles(*subject, roles)
	if err != nil {
		logger.Error(err.Error())
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	return logical.RespondWithStatusCode(&logical.Response{
		Data: map[string]interface{}{
			"effective_roles": effectiveRolesMap,
		},
	}, req, http.StatusOK)
}
