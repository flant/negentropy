package jwtauth

import (
	"context"
	"errors"
	"fmt"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn/multipass"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authz"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn/jwt"
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

	vaultClient, err := b.accessVaultController.APIClient()
	if err != nil {
		logger.Error(fmt.Sprintf("Does not getting vault client %v", err))
		return nil, fmt.Errorf("internal error")
	}

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

	var authenticator authn.Authenticator
	var authSource *model.AuthSource

	logger.Debug("Choice method type")

	switch method.MethodType {
	case model.MethodTypeJWT:
		repo := repos.NewAuthSourceRepo(tnx)
		authSource, err = repo.Get(method.Source)
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

		authenticator = &jwt.Authenticator{
			AuthMethod:   method,
			Logger:       logger.Named("AutheNticator"),
			AuthSource:   authSource,
			JwtValidator: jwtValidator,
		}

	case model.MethodTypeMultipass:
		logger.Debug("It is multipass. Check jwt is enabled")

		enabled, err := b.tokenController.IsEnabled(ctx, req)
		if err != nil {
			return nil, err
		}
		if !enabled {
			logger.Warn("jwt is not enabled. not use multipass login")
			return logical.ErrorResponse("jwt is not enabled. not use multipass login"), nil
		}

		logger.Debug("Jwt is enabled. Get jwt config")

		keys := make([]string, 0)
		jwtConf, err := b.tokenController.GetConfig(ctx, req.Storage)
		if err != nil {
			return nil, err
		}

		logger.Debug("Got jwt config")

		authSource = model.GetMultipassSourceForLogin(jwtConf, keys)

		authenticator = &multipass.Authenticator {
			Logger:       logger.Named("AutheNticator"),
			AuthSource:   authSource,
			MultipassRepo: iam.NewMultipassRepository(tnx),
		}

	default:
		logger.Warn(fmt.Sprintf("unsupported auth method %s", method.MethodType))
		return logical.ErrorResponse("unsupported auth method"), nil
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

	logger.Debug("Start authenticate")

	authnRes, err := authenticator.Authenticate(ctx, d)
	if err != nil {
		logger.Error(fmt.Sprintf("Not authn %v", err))
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	authorizator := authz.Authorizator{
		Logger: logger.Named("AuthoriZator"),

		SaRepo: iam.NewServiceAccountRepository(tnx),
		UserRepo: iam.NewUserRepository(tnx),

		EaRepo: model.NewEntityAliasRepo(tnx),
		EntityRepo: model.NewEntityRepo(tnx),

		MountAccessor: b.accessorGetter,
		IdentityApi: api.NewIdentityAPI(vaultClient, logger.Named("LoginIdentityApi")),
	}

	logger.Debug("Start Authorize")

	authzRes, err := authorizator.Authorize(authnRes, method, authSource)
	if err != nil {
		logger.Error(fmt.Sprintf("Not authz %v", err))
		return logical.ErrorResponse(err.Error()), logical.ErrPermissionDenied
	}

	logger.Debug(fmt.Sprintf("Authorize successful! %s - %s/%s", authzRes.DisplayName, authzRes.EntityID, authzRes.Alias.ID))

	return &logical.Response{
		Auth: authzRes,
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
