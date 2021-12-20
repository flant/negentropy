package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"

	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	authz2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func pathLogin(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `login$`,
		Fields: map[string]*framework.FieldSchema{
			"method": {
				Type:        framework.TypeLowerCaseString,
				Description: "The auth method.",
			},

			"jwt": {
				Type:        framework.TypeString,
				Description: "The signed JWT (or multipass jwt) to validate.",
			},

			"service_account_password_uuid": {
				Type:        framework.TypeString,
				Description: "Service account password uuid. Used for service account password auth",
			},

			"service_account_password_secret": {
				Type:        framework.TypeString,
				Description: "Service account password secret. Used for service account password auth",
			},

			"roles": {
				Type:        framework.TypeSlice,
				Description: "Requested roles",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathLogin,
				Summary:  pathLoginHelpSyn,
			},
			logical.AliasLookaheadOperation: &framework.PathOperation{
				Callback: b.pathLogin,
			},
		},

		HelpSynopsis:    pathLoginHelpSyn,
		HelpDescription: pathLoginHelpDesc,
	}
}

func (b *flantIamAuthBackend) pathLogin(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	logger := b.NamedLogger("Login")

	methodName := d.Get("method").(string)

	if methodName == "" {
		return logical.ErrorResponse("missing method"), nil
	}

	roleClaims, err := getRoleClaims(d)
	if err != nil {
		return nil, fmt.Errorf("%w:parsing roles:%s", consts.ErrInvalidArg, err.Error())
	}

	txn := b.storage.Txn(false)
	repo := repo2.NewAuthMethodRepo(txn)
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return logical.ErrorResponse("method %q could not be found", methodName), nil
	}

	logger.Debug("Checking bound CIDR")
	if len(method.TokenBoundCIDRs) > 0 {
		if req.Connection == nil {
			logger.Warn("token bound CIDRs found but no connection information available for validation")
			return nil, logical.ErrPermissionDenied
		}
		if !cidrutil.RemoteAddrIsOk(req.Connection.RemoteAddr, method.TokenBoundCIDRs) {
			return nil, logical.ErrPermissionDenied
		}
	}

	logger.Debug("Choice method type")
	authenticator, authSource, err := b.authnFactoty.GetAuthenticator(ctx, method, txn)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	logger.Debug("Start authenticate")
	authnRes, err := authenticator.Authenticate(ctx, d)
	if err != nil {
		logger.Error(fmt.Sprintf("Not authn, err: %v", err))
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	authorizator := authz2.NewAutorizator(txn, b.accessVaultProvider, b.accessorGetter, logger)

	logger.Debug("Start Authorize")
	authzRes, err := authorizator.Authorize(authnRes, method, authSource, roleClaims)
	if err != nil {
		logger.Error(fmt.Sprintf("Not authz, err: %v", err))
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	authzRes.InternalData["flantIamAuthMethod"] = method.Name

	logger.Debug(fmt.Sprintf("Authorize successful! %s - %s/%s", authzRes.DisplayName, authzRes.EntityID, authzRes.Alias.ID))

	return &logical.Response{
		Auth: authzRes,
	}, nil
}

func getRoleClaims(d *framework.FieldData) ([]authz2.RoleClaim, error) {
	if roleMaps, ok := d.Get("roles").([]interface{}); ok {
		result := []authz2.RoleClaim{}
		data, err := json.Marshal(roleMaps)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &result)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, nil
}

func (b *flantIamAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := req.Auth.InternalData["flantIamAuthMethod"].(string)
	if methodName == "" {
		return nil, errors.New("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	txn := b.storage.Txn(false)
	repo := repo2.NewAuthMethodRepo(txn)
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("failed to validate authMethodConfig %s during renewal: {{err}}", methodName), err)
	}
	if method == nil {
		return nil, fmt.Errorf("authMethodConfig %s does not exist during renewal", methodName)
	}

	authenticator, _, err := b.authnFactoty.GetAuthenticator(ctx, method, txn)
	if err != nil {
		return nil, err
	}

	can, err := authenticator.CanRenew(req.Auth)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	if !can {
		return logical.ErrorResponse("Can'not prolong renew"), nil
	}

	// TODO autorize user here

	resp := &logical.Response{Auth: req.Auth}
	resp.Auth.TTL = method.TokenTTL
	resp.Auth.MaxTTL = method.TokenMaxTTL
	resp.Auth.Period = method.TokenPeriod

	return resp, nil
}

const (
	pathLoginHelpSyn = `
	Authenticates to Vault using a JWT (or OIDC) token.
	`
	pathLoginHelpDesc = `
Authenticates JWTs.
`
)
