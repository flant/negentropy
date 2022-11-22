package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/certutil"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	jwt2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/jwt"
)

var entityAliasNames = map[string]bool{
	model.EntityAliasNameEmail:          true,
	model.EntityAliasNameFullIdentifier: true,
	model.EntityAliasNameUUID:           true,
}

func pathAuthSource(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `auth_source/` + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeNameString,
				Description: "auth source name",
				Required:    true,
			},
			"oidc_discovery_url": {
				Type:        framework.TypeString,
				Description: `OIDC Discovery URL, without any .well-known component (base path). Cannot be used with "jwks_url" or "jwt_validation_pubkeys".`,
			},
			"oidc_discovery_ca_pem": {
				Type:        framework.TypeString,
				Description: "The CA certificate or chain of certificates, in PEM format, to use to validate connections to the OIDC Discovery URL. If not set, system certificates are used.",
			},
			"oidc_client_id": {
				Type:        framework.TypeString,
				Description: "The OAuth Client ID configured with your OIDC provider.",
			},
			"oidc_client_secret": {
				Type:        framework.TypeString,
				Description: "The OAuth Client Secret configured with your OIDC provider.",
				DisplayAttrs: &framework.DisplayAttributes{
					Sensitive: true,
				},
			},
			"oidc_response_mode": {
				Type:        framework.TypeString,
				Description: "The response mode to be used in the OAuth2 request. Allowed values are 'query' and 'form_post'.",
			},
			"oidc_response_types": {
				Type:        framework.TypeCommaStringSlice,
				Description: "The response types to request. Allowed values are 'code' and 'id_token'. Defaults to 'code'.",
			},
			"jwks_url": {
				Type:        framework.TypeString,
				Description: `JWKS URL to use to authenticate signatures. Cannot be used with "oidc_discovery_url" or "jwt_validation_pubkeys".`,
			},
			"jwks_ca_pem": {
				Type:        framework.TypeString,
				Description: "The CA certificate or chain of certificates, in PEM format, to use to validate connections to the JWKS URL. If not set, system certificates are used.",
			},
			"default_role": {
				Type:        framework.TypeLowerCaseString,
				Description: "The default authMethodConfig to use if none is provided during login. If not set, a authMethodConfig is required during login.",
			},
			"jwt_validation_pubkeys": {
				Type:        framework.TypeCommaStringSlice,
				Description: `A list of PEM-encoded public keys to use to authenticate signatures locally. Cannot be used with "jwks_url" or "oidc_discovery_url".`,
			},
			"jwt_supported_algs": {
				Type:        framework.TypeCommaStringSlice,
				Description: `A list of supported signing algorithms. Defaults to RS256.`,
			},
			"bound_issuer": {
				Type:        framework.TypeString,
				Description: "The value against which to match the 'iss' claim in a JWT. Optional.",
			},
			"provider_config": {
				Type:        framework.TypeMap,
				Description: "Provider-specific configuration. Optional.",
				DisplayAttrs: &framework.DisplayAttributes{
					Name: "Provider Config",
				},
			},
			"namespace_in_state": {
				Type:        framework.TypeBool,
				Description: "Pass namespace in the OIDC state parameter instead of as a separate query parameter. With this setting, the allowed redirect URL(s) in Vault and on the provider side should not contain a namespace query parameter. This means only one redirect URL entry needs to be maintained on the provider side for all vault namespaces that will be authenticating against it. Defaults to true for new configs.",
				DisplayAttrs: &framework.DisplayAttributes{
					Name:  "Namespace in OIDC state",
					Value: true,
				},
			},

			"entity_alias_name": {
				Type: framework.TypeString,
				Description: fmt.Sprintf("entity alias name source. may be  '%s', '%s',  or '%s'.",
					model.EntityAliasNameEmail, model.EntityAliasNameFullIdentifier, model.EntityAliasNameUUID),
				Required: true,
			},

			"allow_service_accounts": {
				Type:        framework.TypeBool,
				Default:     false,
				Description: "Allow create entity aliases for services accounts fot this source",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathAuthSourceWrite,
				Summary:  "Write authentication source name passed name.",
			},

			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathAuthSourceRead,
				Summary:  "Read authentication source.",
			},

			logical.DeleteOperation: &framework.PathOperation{
				Callback:    b.pathAuthSourceDelete,
				Summary:     "Delete authentication source.",
				Description: confHelpDesc,
			},
		},

		HelpSynopsis:    confHelpSyn,
		HelpDescription: confHelpDesc,
	}
}

func pathAuthSourceList(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "auth_source/?",
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback:    b.pathAuthSourceList,
				Summary:     "List authentication source.",
				Description: confHelpDesc,
			},
		},
	}
}

func (b *flantIamAuthBackend) pathAuthSourceRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	sourceName, errorResp := nameFromRequest(d)
	if errorResp != nil {
		return errorResp, nil
	}

	txn := b.storage.Txn(false)
	repo := repo2.NewAuthSourceRepo(txn)

	config, err := repo.Get(sourceName)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"oidc_discovery_url":     config.OIDCDiscoveryURL,
			"oidc_discovery_ca_pem":  config.OIDCDiscoveryCAPEM,
			"oidc_client_id":         config.OIDCClientID,
			"oidc_response_mode":     config.OIDCResponseMode,
			"oidc_response_types":    config.OIDCResponseTypes,
			"default_role":           config.DefaultRole,
			"jwt_validation_pubkeys": config.JWTValidationPubKeys,
			"jwt_supported_algs":     config.JWTSupportedAlgs,
			"jwks_url":               config.JWKSURL,
			"jwks_ca_pem":            config.JWKSCAPEM,
			"bound_issuer":           config.BoundIssuer,
			"namespace_in_state":     config.NamespaceInState,
			"entity_alias_name":      config.EntityAliasName,
			"allow_service_accounts": config.AllowServiceAccounts,
		},
	}

	return resp, nil
}

func (b *flantIamAuthBackend) pathAuthSourceWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	sourceName, errorResp := nameFromRequest(d)
	if errorResp != nil {
		return errorResp, nil
	}

	sourceForStore := &model.AuthSource{
		Name: sourceName,

		OIDCDiscoveryURL:     d.Get("oidc_discovery_url").(string),
		OIDCDiscoveryCAPEM:   d.Get("oidc_discovery_ca_pem").(string),
		OIDCClientID:         d.Get("oidc_client_id").(string),
		OIDCClientSecret:     d.Get("oidc_client_secret").(string),
		OIDCResponseMode:     d.Get("oidc_response_mode").(string),
		OIDCResponseTypes:    d.Get("oidc_response_types").([]string),
		JWKSURL:              d.Get("jwks_url").(string),
		JWKSCAPEM:            d.Get("jwks_ca_pem").(string),
		DefaultRole:          d.Get("default_role").(string),
		JWTValidationPubKeys: d.Get("jwt_validation_pubkeys").([]string),
		JWTSupportedAlgs:     d.Get("jwt_supported_algs").([]string),
		BoundIssuer:          d.Get("bound_issuer").(string),
		EntityAliasName:      d.Get("entity_alias_name").(string),
		AllowServiceAccounts: d.Get("allow_service_accounts").(bool),
	}

	if _, ok := entityAliasNames[sourceForStore.EntityAliasName]; !ok {
		return logical.ErrorResponse(fmt.Sprintf("incorrect entity_alias_name %v", sourceForStore.EntityAliasName)), nil
	}

	if sourceForStore.EntityAliasName == model.EntityAliasNameEmail && sourceForStore.AllowServiceAccounts {
		return logical.ErrorResponse("conflict values for entity_alias_name and allow_service_accounts"), nil
	}

	// Check if the sourceForStore already exists, to determine if this is a create or
	// an update, since req.Operation is always 'update' in this handler, and
	// there's no existence check defined.

	txn := b.storage.Txn(true)
	defer txn.Abort()

	repo := repo2.NewAuthSourceRepo(txn)
	existingSource, err := repo.Get(sourceName)
	if err != nil {
		return nil, err
	}

	nsInState, ok := d.GetOk("namespace_in_state")
	switch {
	case ok:
		sourceForStore.NamespaceInState = nsInState.(bool)
	case existingSource == nil:
		// new configs default to true
		sourceForStore.NamespaceInState = true
	default:
		// maintain the existing value
		sourceForStore.NamespaceInState = existingSource.NamespaceInState
	}

	// Run checks on values
	methodCount := 0
	if sourceForStore.OIDCDiscoveryURL != "" {
		methodCount++
	}
	if len(sourceForStore.JWTValidationPubKeys) != 0 {
		methodCount++
	}
	if sourceForStore.JWKSURL != "" {
		methodCount++
	}

	switch {
	case methodCount != 1:
		return logical.ErrorResponse("exactly one of 'jwt_validation_pubkeys', 'jwks_url' or 'oidc_discovery_url' must be set"), nil

	case sourceForStore.OIDCClientID != "" && sourceForStore.OIDCClientSecret == "",
		sourceForStore.OIDCClientID == "" && sourceForStore.OIDCClientSecret != "":
		return logical.ErrorResponse("both 'oidc_client_id' and 'oidc_client_secret' must be set for OIDC"), nil

	case sourceForStore.OIDCDiscoveryURL != "":
		var err error
		if sourceForStore.OIDCClientID != "" && sourceForStore.OIDCClientSecret != "" {
			_, err = b.createProvider(sourceForStore)
		} else {
			_, err = jwt.NewOIDCDiscoveryKeySet(ctx, sourceForStore.OIDCDiscoveryURL, sourceForStore.OIDCDiscoveryCAPEM)
		}
		if err != nil {
			return logical.ErrorResponse("error checking oidc discovery URL: %s", err.Error()), nil
		}

	case sourceForStore.OIDCClientID != "" && sourceForStore.OIDCDiscoveryURL == "":
		return logical.ErrorResponse("'oidc_discovery_url' must be set for OIDC"), nil

	case sourceForStore.JWKSURL != "":
		keyset, err := jwt.NewJSONWebKeySet(ctx, sourceForStore.JWKSURL, sourceForStore.JWKSCAPEM)
		if err != nil {
			return logical.ErrorResponse(errwrap.Wrapf("error checking jwks_ca_pem: {{err}}", err).Error()), nil
		}

		// Try to verify a correctly formatted JWT. The signature will fail to match, but other
		// errors with fetching the remote keyset should be reported.
		testJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.Hf3E3iCHzqC5QIQ0nCqS1kw78IiQTRVzsLTuKoDIpdk"
		_, err = keyset.VerifySignature(ctx, testJWT)
		if err == nil {
			err = errors.New("unexpected verification of JWT")
		}

		if !strings.Contains(err.Error(), "failed to verify id token signature") {
			return logical.ErrorResponse(errwrap.Wrapf("error checking jwks URL: {{err}}", err).Error()), nil
		}

	case len(sourceForStore.JWTValidationPubKeys) != 0:
		for _, v := range sourceForStore.JWTValidationPubKeys {
			if _, err := certutil.ParsePublicKeyPEM([]byte(v)); err != nil {
				return logical.ErrorResponse(errwrap.Wrapf("error parsing public key: {{err}}", err).Error()), nil
			}
		}

	default:
		return nil, errors.New("unknown condition")
	}

	// NOTE: the OIDC lib states that if nothing is passed into its sourceForStore, it
	// defaults to "RS256". So in the case of a zero value here it won't
	// default to e.g. "none".
	if err := jwt.SupportedSigningAlgorithm(jwt2.ToAlg(sourceForStore.JWTSupportedAlgs)...); err != nil {
		return logical.ErrorResponse("invalid jwt_supported_algs: %s", err), nil
	}

	// Validate response_types
	if !strutil.StrListSubset([]string{model.ResponseTypeCode, model.ResponseTypeIDToken}, sourceForStore.OIDCResponseTypes) {
		return logical.ErrorResponse("invalid response_types %v. 'code' and 'id_token' are allowed", sourceForStore.OIDCResponseTypes), nil
	}

	// Validate response_mode
	switch sourceForStore.OIDCResponseMode {
	case "", model.ResponseModeQuery:
		if sourceForStore.HasType(model.ResponseTypeIDToken) {
			return logical.ErrorResponse("query response_mode may not be used with an id_token response_type"), nil
		}
	case model.ResponseModeFormPost:
	default:
		return logical.ErrorResponse("invalid response_mode: %q", sourceForStore.OIDCResponseMode), nil
	}

	if err := repo.Put(sourceForStore); err != nil {
		return nil, err
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	b.reset()

	return nil, nil
}

func (b *flantIamAuthBackend) pathAuthSourceList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	txn := b.storage.Txn(false)
	repo := repo2.NewAuthSourceRepo(txn)

	var sourcesNames []string
	err := repo.Iter(false, func(s *model.AuthSource) (bool, error) {
		sourcesNames = append(sourcesNames, s.Name)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(sourcesNames), nil
}

func (b *flantIamAuthBackend) pathAuthSourceDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	sourceName, errorResp := nameFromRequest(d)
	if errorResp != nil {
		return errorResp, nil
	}

	txn := b.storage.Txn(true)
	defer txn.Abort()

	repo := repo2.NewAuthSourceRepo(txn)

	err := repo.Delete(sourceName)
	if err != nil {
		switch {
		case errors.Is(err, repo2.ErrSourceUsingInMethods):
			return logical.ErrorResponse("%v", err), nil
		case errors.Is(err, repo2.ErrSourceNotFound):
			return logical.ErrorResponse("source not found"), nil
		}

		return nil, err
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *flantIamAuthBackend) createProvider(source *model.AuthSource) (*oidc.Provider, error) {
	supportedSigAlgs := make([]oidc.Alg, len(source.JWTSupportedAlgs))
	for i, a := range source.JWTSupportedAlgs {
		supportedSigAlgs[i] = oidc.Alg(a)
	}

	if len(supportedSigAlgs) == 0 {
		supportedSigAlgs = []oidc.Alg{oidc.RS256}
	}

	c, err := oidc.NewConfig(source.OIDCDiscoveryURL, source.OIDCClientID,
		oidc.ClientSecret(source.OIDCClientSecret), supportedSigAlgs, []string{},
		oidc.WithProviderCA(source.OIDCDiscoveryCAPEM))
	if err != nil {
		return nil, errwrap.Wrapf("error creating provider: {{err}}", err)
	}

	provider, err := oidc.NewProvider(c)
	if err != nil {
		return nil, errwrap.Wrapf("error creating provider with given values: {{err}}", err)
	}

	return provider, nil
}

const (
	confHelpSyn = `
Configures the JWT authentication backend.
`
	confHelpDesc = `
The JWT authentication backend validates JWTs (or OIDC) using the configured
credentials. If using OIDC Discovery, the URL must be provided, along
with (optionally) the CA cert to use for the connection. If performing JWT
validation locally, a set of public keys must be provided.
`
)
