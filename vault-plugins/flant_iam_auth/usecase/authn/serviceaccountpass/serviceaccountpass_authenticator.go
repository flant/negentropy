package serviceaccountpass

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
)

type Authenticator struct {
	ServiceAccountPasswordRepo *repo.ServiceAccountPasswordRepository

	AuthMethod *model.AuthMethod

	Logger hclog.Logger
}

func (a *Authenticator) Authenticate(ctx context.Context, d *framework.FieldData) (*authn.Result, error) {
	a.Logger.Debug("Start authn service_account_password")
	passwordID := d.Get("service_account_password_uuid").(string)
	password := d.Get("service_account_password_secret").(string)
	serviceAccountPassword, err := a.ServiceAccountPasswordRepo.GetByID(passwordID)
	if err != nil {
		return nil, err
	}
	if password != serviceAccountPassword.Secret {
		return nil, fmt.Errorf("wrong secret")
	}

	return &authn.Result{
		UUID:         serviceAccountPassword.OwnerUUID,
		Metadata:     map[string]string{},
		GroupAliases: make([]string, 0),
		InternalData: map[string]interface{}{
			"service_account_password": map[string]interface{}{
				"service_account_password_uuid": passwordID,
				// TODO add staff for renew checking
			},
		},
	}, nil
}

func (a *Authenticator) CanRenew(vaultAuth *logical.Auth) (bool, error) {
	// TODO
	return false, nil
}
