package backend

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
)

func pathMultipassOwner(b *flantIamAuthBackend) *framework.Path {
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
	}
	msg := fmt.Sprintf("unexpected subjectType: `%s`", subjectType)
	logger.Debug(msg)
	return responseErrMessage(req, err.Error(), http.StatusInternalServerError)
}
