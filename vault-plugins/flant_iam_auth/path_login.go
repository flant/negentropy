package jwtauth

import (
	"context"
	"errors"
	"fmt"


	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repos "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
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
				Description: "The signed JWT to validate.",
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
	methodName := d.Get("method").(string)

	if methodName == "" {
		return logical.ErrorResponse("missing method"), nil
	}

	tnx := b.storage.Txn(false)
	repo := repos.NewAuthMethodRepo(tnx)
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return logical.ErrorResponse("method %q could not be found", methodName), nil
	}

	var authenticator Authenticator

	switch method.MethodType {
	case model.MethodTypeJWT:
		repo := repos.NewAuthSourceRepo(tnx)
		authSource, err := repo.Get(method.Source)
		if err != nil {
			return nil, err
		}
		if authSource == nil {
			return logical.ErrorResponse("not found auth method"), nil
		}

		jwtValidator, err := b.jwtValidator(methodName, authSource)
		if err != nil {
			return nil, err
		}

		authenticator = &JwtAuthenticator{
			authMethod:   method,
			methodName:   methodName,
			logger:       b.Logger(),
			authSource:   authSource,
			jwtValidator: jwtValidator,
		}
	default:
		return logical.ErrorResponse("unsupported auth method"), nil
	}

	if len(method.TokenBoundCIDRs) > 0 {
		if req.Connection == nil {
			b.Logger().Warn("token bound CIDRs found but no connection information available for validation")
			return nil, logical.ErrPermissionDenied
		}
		if !cidrutil.RemoteAddrIsOk(req.Connection.RemoteAddr, method.TokenBoundCIDRs) {
			return nil, logical.ErrPermissionDenied
		}
	}

	auth, err := authenticator.Auth(ctx, d)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	method.PopulateTokenAuth(auth)

	auth.InternalData["flantIamAuthMethod"] = methodName

	return &logical.Response{
		Auth: auth,
	}, nil
}

func (b *flantIamAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := req.Auth.InternalData["flantIamAuthMethod"].(string)
	if methodName == "" {
		return nil, errors.New("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	tnx := b.storage.Txn(false)
	repo := repos.NewAuthMethodRepo(tnx)
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("failed to validate authMethodConfig %s during renewal: {{err}}", methodName), err)
	}
	if method == nil {
		return nil, fmt.Errorf("authMethodConfig %s does not exist during renewal", methodName)
	}

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
