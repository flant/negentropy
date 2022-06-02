package backend

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func pathVSTOwner(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `vst_owner$`,
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.vstOwner,
				Summary:  pathLoginHelpSyn,
			},
		},
		HelpSynopsis: "Provide info about owner of vault session token (if it issued for user or service_account of negentropy)",
	}
}

func (b *flantIamAuthBackend) vstOwner(ctx context.Context, req *logical.Request,
	d *framework.FieldData) (*logical.Response, error) {
	entityIDOwner, err := b.entityIDResolver.RevealEntityIDOwner(req.EntityID, b.storage.Txn(false), req.Storage)
	if errors.Is(err, consts.ErrNotFound) {
		return logical.RespondWithStatusCode(nil, req, http.StatusNotFound) //nolint:errCheck
	}
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	return logical.RespondWithStatusCode(&logical.Response{
		Data: map[string]interface{}{
			entityIDOwner.OwnerType: entityIDOwner.Owner,
		},
	}, req, http.StatusOK)
}
