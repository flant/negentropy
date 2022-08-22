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

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	authz2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func pathLogin(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `login$`,
		Fields: map[string]*framework.FieldSchema{
			"method": {
				Type:        framework.TypeLowerCaseString,
				Description: "The auth method.",
				Required:    true,
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
		return backentutils.ResponseErr(req, fmt.Errorf("%w:missing method", consts.ErrInvalidArg))
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
	authenticator, authSource, err := b.authnFactory.GetAuthenticator(ctx, method, txn)
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

	authzRes.Renewable = true
	return &logical.Response{
		Auth: authzRes,
	}, nil
}

func getRoleClaims(d *framework.FieldData) ([]model.RoleClaim, error) {
	if roleMaps, ok := d.Get("roles").([]interface{}); ok {
		result := []model.RoleClaim{}
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
	logger := b.NamedLogger("renew")
	logger.Debug(fmt.Sprintf("TODO REMOVE! enter renew, %#v", req.Auth))

	methodName := req.Auth.InternalData["flantIamAuthMethod"].(string)
	if methodName == "" {
		return nil, errors.New("failed to fetch auth_method during renewal")
	}

	// Ensure that the auth_method still exists.
	txn := b.storage.Txn(false)
	method, err := repo2.NewAuthMethodRepo(txn).Get(methodName)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("failed to validate authMethodConfig %s during renewal: {{err}}", methodName), err)
	}
	if method == nil {
		return nil, fmt.Errorf("authMethodConfig %s does not exist during renewal", methodName)
	}

	authenticator, _, err := b.authnFactory.GetAuthenticator(ctx, method, txn)
	if err != nil {
		return nil, err
	}

	can, err := authenticator.CanRenew(req.Auth)
	if err != nil {
		logger.Error(err.Error())
		return logical.ErrorResponse(err.Error()), nil
	}

	if !can {
		txt := "Can't prolong renew"
		logger.Debug(txt)
		return logical.ErrorResponse(txt), nil
	}

	authorizator := authz2.NewAutorizator(txn, b.accessVaultProvider, b.accessorGetter, logger)

	logger.Debug("Start renew")
	rawSubject := req.Auth.InternalData["subject"]
	logger.Debug(fmt.Sprintf("%#v", rawSubject))
	subjectData, _ := req.Auth.InternalData["subject"].(map[string]interface{})
	subject := authz2.MakeSubject(subjectData)
	authzRes, err := authorizator.Renew(method, req.Auth, txn, subject)
	if err != nil {
		logger.Error(fmt.Sprintf("Not renew authz, err: %v", err))
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}
	return &logical.Response{
		Auth: authzRes,
	}, nil
}

const (
	pathLoginHelpSyn = `
	Authenticates to Vault using a JWT (or OIDC) token, or service_account password
	`
	pathLoginHelpDesc = `
Authenticates to Vault using a JWT (or OIDC) token, or service_account password
`
)
