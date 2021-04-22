package jwtauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"
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
	authMethodName := d.Get("method").(string)

	if authMethodName == "" {
		return logical.ErrorResponse("missing authMethodConfig"), nil
	}

	authMethod, err := b.authMethodForRequest(ctx, req, authMethodName)
	if err != nil {
		return nil, err
	}
	if authMethod == nil {
		return logical.ErrorResponse(" %q could not be found", authMethodName), nil
	}

	var authenticator Authenticator

	switch authMethod.MethodType {
	case methodTypeJWT:
		authSource, err := b.authSource(ctx, req, authMethod.Source)
		if err != nil {
			return nil, err
		}
		if authSource == nil {
			return logical.ErrorResponse("not found auth method"), nil
		}

		jwtValidator, err := b.jwtValidator(authMethodName, authSource)
		if err != nil {
			return nil, err
		}

		authenticator = &JwtAuthenticator{
			authMethod:   authMethod,
			logger:       b.Logger(),
			authSource:   authSource,
			jwtValidator: jwtValidator,
		}
	default:
		return logical.ErrorResponse("unsupported auth method"), nil
	}

	if len(authMethod.TokenBoundCIDRs) > 0 {
		if req.Connection == nil {
			b.Logger().Warn("token bound CIDRs found but no connection information available for validation")
			return nil, logical.ErrPermissionDenied
		}
		if !cidrutil.RemoteAddrIsOk(req.Connection.RemoteAddr, authMethod.TokenBoundCIDRs) {
			return nil, logical.ErrPermissionDenied
		}
	}

	auth, err := authenticator.Auth(ctx, d)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	authMethod.PopulateTokenAuth(auth)

	return &logical.Response{
		Auth: auth,
	}, nil
}

func (b *flantIamAuthBackend) pathLoginRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	authMethodName := req.Auth.InternalData["authMethodConfig"].(string)
	if authMethodName == "" {
		return nil, errors.New("failed to fetch role_name during renewal")
	}

	// Ensure that the Role still exists.
	authMethod, err := b.authMethod(ctx, req.Storage, authMethodName)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("failed to validate authMethodConfig %s during renewal: {{err}}", authMethodName), err)
	}
	if authMethod == nil {
		return nil, fmt.Errorf("authMethodConfig %s does not exist during renewal", authMethodName)
	}

	resp := &logical.Response{Auth: req.Auth}
	resp.Auth.TTL = authMethod.TokenTTL
	resp.Auth.MaxTTL = authMethod.TokenMaxTTL
	resp.Auth.Period = authMethod.TokenPeriod
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
