package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/tokenutil"
	"github.com/hashicorp/vault/sdk/logical"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repos "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

var reservedMetadata = []string{"flantIamAuthMethod"}

var authMethodTypes = []string{model.MethodTypeOwn, model.MethodTypeJWT, model.MethodTypeOIDC, model.MethodTypeSAPassword}

func pathAuthMethodList(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: "auth_method/?",
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback:    b.pathRoleList,
				Summary:     strings.TrimSpace(roleHelp["authMethodConfig-list"][0]),
				Description: strings.TrimSpace(roleHelp["authMethodConfig-list"][1]),
			},
		},
		HelpSynopsis:    strings.TrimSpace(roleHelp["authMethodConfig-list"][0]),
		HelpDescription: strings.TrimSpace(roleHelp["authMethodConfig-list"][1]),
	}
}

// pathRole returns the path configurations for the CRUD operations on roles
func pathAuthMethod(b *flantIamAuthBackend) *framework.Path {
	p := &framework.Path{
		Pattern: "auth_method/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the authMethodConfig.",
			},
			"method_type": {
				Type: framework.TypeString,
				Description: fmt.Sprintf("Type of the authMethodConfig, either '%s', '%s', '%s' or '%s'.",
					model.MethodTypeJWT, model.MethodTypeOIDC, model.MethodTypeSAPassword, model.MethodTypeOwn),
				Required: true,
			},
			"source": {
				Type: framework.TypeString,
				Description: fmt.Sprintf("authentification source for method thypes '%s' and '%s'.",
					model.MethodTypeJWT, model.MethodTypeOIDC),
			},

			"expiration_leeway": {
				Type: framework.TypeSignedDurationSecond,
				Description: `Duration in seconds of leeway when validating expiration of a token to account for clock skew. 
Defaults to 150 (2.5 minutes) if set to 0 and can be disabled if set to -1.`,
				Default: model.ClaimDefaultLeeway,
			},
			"not_before_leeway": {
				Type: framework.TypeSignedDurationSecond,
				Description: `Duration in seconds of leeway when validating not before values of a token to account for clock skew. 
Defaults to 150 (2.5 minutes) if set to 0 and can be disabled if set to -1.`,
				Default: model.ClaimDefaultLeeway,
			},
			"clock_skew_leeway": {
				Type: framework.TypeSignedDurationSecond,
				Description: `Duration in seconds of leeway when validating all claims to account for clock skew. 
Defaults to 60 (1 minute) if set to 0 and can be disabled if set to -1.`,
				Default: jwt.DefaultLeeway,
			},
			"bound_subject": {
				Type:        framework.TypeString,
				Description: `The 'sub' claim that is valid for login. Optional.`,
			},
			"bound_audiences": {
				Type:        framework.TypeCommaStringSlice,
				Description: `Comma-separated list of 'aud' claims that are valid for login; any match is sufficient`,
			},
			"bound_claims_type": {
				Type:        framework.TypeString,
				Description: `How to interpret values in the map of claims/values (which must match for login): allowed values are 'string' or 'glob'`,
				Default:     model.BoundClaimsTypeString,
			},
			"bound_claims": {
				Type:        framework.TypeMap,
				Description: `Map of claims/values which must match for login`,
			},
			"claim_mappings": {
				Type:        framework.TypeKVPairs,
				Description: `Mappings of claims (key) that will be copied to a metadata field (value)`,
			},
			"user_claim": {
				Type:        framework.TypeString,
				Description: `The claim to use for the Identity entity alias name`,
			},
			"groups_claim": {
				Type:        framework.TypeString,
				Description: `The claim to use for the Identity group alias names`,
			},
			"oidc_scopes": {
				Type:        framework.TypeCommaStringSlice,
				Description: `Comma-separated list of OIDC scopes`,
			},
			"allowed_redirect_uris": {
				Type:        framework.TypeCommaStringSlice,
				Description: `Comma-separated list of allowed values for redirect_uri`,
			},
			"verbose_oidc_logging": {
				Type: framework.TypeBool,
				Description: `Log received OIDC tokens and claims when debug-level logging is active. 
Not recommended in production since sensitive information may be present 
in OIDC responses.`,
			},
			"max_age": {
				Type: framework.TypeDurationSecond,
				Description: `Specifies the allowable elapsed time in seconds since the last time the 
user was actively authenticated.`,
			},
		},
		ExistenceCheck: b.pathAuthMethodExistenceCheck,
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathAuthMethodRead,
				Summary:  "Read an existing authMethodConfig.",
			},

			logical.UpdateOperation: &framework.PathOperation{
				Callback:    b.pathAuthMethodCreateUpdate,
				Summary:     strings.TrimSpace(roleHelp["authMethodConfig"][0]),
				Description: strings.TrimSpace(roleHelp["authMethodConfig"][1]),
			},

			logical.CreateOperation: &framework.PathOperation{
				Callback:    b.pathAuthMethodCreateUpdate,
				Summary:     strings.TrimSpace(roleHelp["authMethodConfig"][0]),
				Description: strings.TrimSpace(roleHelp["authMethodConfig"][1]),
			},

			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathAuthMethodDelete,
				Summary:  "Delete an existing authMethodConfig.",
			},
		},
		HelpSynopsis:    strings.TrimSpace(roleHelp["authMethodConfig"][0]),
		HelpDescription: strings.TrimSpace(roleHelp["authMethodConfig"][1]),
	}

	tokenutil.AddTokenFields(p.Fields)
	return p
}

// pathAuthMethodExistenceCheck returns whether the authMethodConfig with the given name exists or not.
func (b *flantIamAuthBackend) pathAuthMethodExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	methodName := data.Get("name").(string)

	tnx := b.storage.Txn(false)
	repo := repos.NewAuthMethodRepo(tnx)
	method, err := repo.Get(methodName)
	if err != nil {
		return false, err
	}
	return method != nil, nil
}

// pathRoleList is used to list all the Roles registered with the backend.
func (b *flantIamAuthBackend) pathRoleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tnx := b.storage.Txn(false)
	repo := repos.NewAuthMethodRepo(tnx)

	var methodsNames []string
	err := repo.Iter(func(s *model.AuthMethod) (bool, error) {
		methodsNames = append(methodsNames, s.Name)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(methodsNames), nil
}

// pathAuthMethodRead grabs a read lock and reads the options set on the authMethodConfig from the storage
func (b *flantIamAuthBackend) pathAuthMethodRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := data.Get("name").(string)
	if methodName == "" {
		return logical.ErrorResponse("missing name"), nil
	}

	tnx := b.storage.Txn(false)
	repo := repos.NewAuthMethodRepo(tnx)
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, nil
	}

	// Create a map of data to be returned
	d := map[string]interface{}{
		"role_type":             method.MethodType,
		"bound_audiences":       method.BoundAudiences,
		"bound_subject":         method.BoundSubject,
		"bound_claims_type":     method.BoundClaimsType,
		"bound_claims":          method.BoundClaims,
		"claim_mappings":        method.ClaimMappings,
		"user_claim":            method.UserClaim,
		"groups_claim":          method.GroupsClaim,
		"allowed_redirect_uris": method.AllowedRedirectURIs,
		"oidc_scopes":           method.OIDCScopes,
		"verbose_oidc_logging":  method.VerboseOIDCLogging,
		"max_age":               int64(method.MaxAge.Seconds()),
	}

	if method.MethodType == model.MethodTypeOwn {
		d["expiration_leeway"] = int64(method.ExpirationLeeway.Seconds())
		d["not_before_leeway"] = int64(method.NotBeforeLeeway.Seconds())
		d["clock_skew_leeway"] = int64(method.ClockSkewLeeway.Seconds())
	}

	if method.MethodType == model.MethodTypeOwn || method.MethodType == model.MethodTypeSAPassword {
		method.PopulateTokenData(d)
	}

	return &logical.Response{
		Data: d,
	}, nil
}

// pathAuthMethodDelete removes the authMethodConfig from storage
func (b *flantIamAuthBackend) pathAuthMethodDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := data.Get("name").(string)
	if methodName == "" {
		return logical.ErrorResponse("authMethodConfig name required"), nil
	}

	tnx := b.storage.Txn(true)
	defer tnx.Abort()

	repo := repos.NewAuthMethodRepo(tnx)

	err := repo.Delete(methodName)
	if err != nil {
		return nil, err
	}

	err = tnx.Commit()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// pathAuthMethodCreateUpdate registers a new authMethodConfig with the backend or updates the options
// of an existing authMethodConfig
func (b *flantIamAuthBackend) pathAuthMethodCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := data.Get("name").(string)
	if methodName == "" {
		return logical.ErrorResponse("missing auth method name"), nil
	}

	tnx := b.storage.Txn(true)
	defer tnx.Abort()

	repo := repos.NewAuthMethodRepo(tnx)
	// Check if the auth already exists
	method, err := repo.Get(methodName)
	if err != nil {
		return nil, err
	}

	// Create a new entry object if this is a CreateOperation
	if method == nil {
		if req.Operation == logical.UpdateOperation {
			return nil, errors.New("auth method entry not found during update operation")
		}
		method = new(model.AuthMethod)
		method.UUID = utils.UUID()
		method.Name = methodName
	}

	methodType := data.Get("method_type").(string)
	if methodType == "" {
		return logical.ErrorResponse("missing method_type"), nil
	}

	isCorrectMethod := false
	for _, m := range authMethodTypes {
		if methodType == m {
			isCorrectMethod = true
			break
		}
	}

	if !isCorrectMethod {
		return logical.ErrorResponse("invalid 'method_type': %s", methodType), nil
	}

	method.MethodType = methodType

	if methodType == model.MethodTypeJWT || methodType == model.MethodTypeOIDC {
		sourceName := data.Get("source").(string)
		if sourceName == "" {
			return logical.ErrorResponse("missing source"), nil
		}

		source, err := repos.NewAuthSourceRepo(tnx).Get(sourceName)
		if err != nil {
			return nil, err
		}

		if source == nil {
			return logical.ErrorResponse(fmt.Sprintf("'%s': auth source not found", sourceName)), nil
		}

		authType := source.AuthType()
		if methodType == model.MethodTypeJWT && !(authType == model.StaticKeys || authType == model.JWKS) {
			return logical.ErrorResponse(fmt.Sprintf("incorrect source '%s': need jwt based source", sourceName)), nil
		}

		if methodType == model.MethodTypeOIDC && !(authType == model.OIDCFlow || authType == model.OIDCDiscovery) {
			return logical.ErrorResponse(fmt.Sprintf("incorrect source '%s': need OIDC based source", sourceName)), nil
		}

		method.Source = sourceName
	}

	if err := method.ParseTokenFields(req, data); err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrInvalidRequest
	}

	if method.TokenPeriod > b.System().MaxLeaseTTL() {
		return logical.ErrorResponse(fmt.Sprintf("'period' of '%q' is greater than the backend's maximum lease TTL of '%q'", method.TokenPeriod.String(), b.System().MaxLeaseTTL().String())), nil
	}

	if methodType == model.MethodTypeOwn {
		if tokenExpLeewayRaw, ok := data.GetOk("expiration_leeway"); ok {
			method.ExpirationLeeway = time.Duration(tokenExpLeewayRaw.(int)) * time.Second
		}

		if tokenNotBeforeLeewayRaw, ok := data.GetOk("not_before_leeway"); ok {
			method.NotBeforeLeeway = time.Duration(tokenNotBeforeLeewayRaw.(int)) * time.Second
		}

		if tokenClockSkewLeeway, ok := data.GetOk("clock_skew_leeway"); ok {
			method.ClockSkewLeeway = time.Duration(tokenClockSkewLeeway.(int)) * time.Second
		}
	}

	if boundAudiences, ok := data.GetOk("bound_audiences"); ok {
		method.BoundAudiences = boundAudiences.([]string)
	}

	if boundSubject, ok := data.GetOk("bound_subject"); ok {
		method.BoundSubject = boundSubject.(string)
	}

	if verboseOIDCLoggingRaw, ok := data.GetOk("verbose_oidc_logging"); ok {
		method.VerboseOIDCLogging = verboseOIDCLoggingRaw.(bool)
	}

	if maxAgeRaw, ok := data.GetOk("max_age"); ok {
		method.MaxAge = time.Duration(maxAgeRaw.(int)) * time.Second
	}

	boundClaimsType := data.Get("bound_claims_type").(string)
	switch boundClaimsType {
	case model.BoundClaimsTypeString, model.BoundClaimsTypeGlob:
		method.BoundClaimsType = boundClaimsType
	default:
		return logical.ErrorResponse("invalid 'bound_claims_type': %s", boundClaimsType), nil
	}

	if boundClaimsRaw, ok := data.GetOk("bound_claims"); ok {
		method.BoundClaims = boundClaimsRaw.(map[string]interface{})

		if boundClaimsType == model.BoundClaimsTypeGlob {
			// Check that the claims are all strings
			for _, claimValues := range method.BoundClaims {
				claimsValuesList, ok := normalizeList(claimValues)

				if !ok {
					return logical.ErrorResponse("claim is not a string or list: %v", claimValues), nil
				}

				for _, claimValue := range claimsValuesList {
					if _, ok := claimValue.(string); !ok {
						return logical.ErrorResponse("claim is not a string: %v", claimValue), nil
					}
				}
			}
		}
	}

	if claimMappingsRaw, ok := data.GetOk("claim_mappings"); ok {
		claimMappings := claimMappingsRaw.(map[string]string)

		// sanity check mappings for duplicates and collision with reserved names
		targets := make(map[string]bool)
		for _, metadataKey := range claimMappings {
			if strutil.StrListContains(reservedMetadata, metadataKey) {
				return logical.ErrorResponse("metadata key %q is reserved and may not be a mapping destination", metadataKey), nil
			}

			if targets[metadataKey] {
				return logical.ErrorResponse("multiple keys are mapped to metadata key %q", metadataKey), nil
			}
			targets[metadataKey] = true
		}

		method.ClaimMappings = claimMappings
	}

	if userClaim, ok := data.GetOk("user_claim"); ok {
		method.UserClaim = userClaim.(string)
	}
	if method.UserClaim == "" {
		return logical.ErrorResponse("a user claim must be defined on the authMethodConfig"), nil
	}

	if groupsClaim, ok := data.GetOk("groups_claim"); ok {
		method.GroupsClaim = groupsClaim.(string)
	}

	if oidcScopes, ok := data.GetOk("oidc_scopes"); ok {
		method.OIDCScopes = oidcScopes.([]string)
	}

	if allowedRedirectURIs, ok := data.GetOk("allowed_redirect_uris"); ok {
		method.AllowedRedirectURIs = allowedRedirectURIs.([]string)
	}

	if method.MethodType == model.MethodTypeOIDC && len(method.AllowedRedirectURIs) == 0 {
		return logical.ErrorResponse(
			"'allowed_redirect_uris' must be set if 'method_type' is 'oidc' or unspecified."), nil
	}

	// OIDC verification will enforce that the audience match the configured client_id.
	// For other methods, require at least one bound constraint.
	if methodType == model.MethodTypeJWT {
		if len(method.BoundAudiences) == 0 &&
			len(method.TokenBoundCIDRs) == 0 &&
			method.BoundSubject == "" &&
			len(method.BoundClaims) == 0 {
			return logical.ErrorResponse("must have at least one bound constraint when creating/updating a authMethod"), nil
		}
	}

	// Check that the TTL value provided is less than the MaxTTL.
	// Sanitizing the TTL and MaxTTL is not required now and can be performed
	// at credential issue time.
	if method.TokenMaxTTL > 0 && method.TokenTTL > method.TokenMaxTTL {
		return logical.ErrorResponse("ttl should not be greater than max ttl"), nil
	}

	resp := &logical.Response{}
	if method.TokenMaxTTL > b.System().MaxLeaseTTL() {
		resp.AddWarning("token max ttl is greater than the system or backend mount's maximum TTL value; issued tokens' max TTL value will be truncated")
	}

	if method.VerboseOIDCLogging {
		resp.AddWarning(`verbose_oidc_logging has been enabled for this authMethodConfig. ` +
			`This is not recommended in production since sensitive information ` +
			`may be present in OIDC responses.`)
	}

	err = repo.Put(method)
	if err != nil {
		return nil, err
	}

	err = tnx.Commit()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// roleStorageEntry stores all the options that are set on an authMethodConfig
var roleHelp = map[string][2]string{
	"authMethodConfig-list": {
		"Lists all the roles registered with the backend.",
		"The list will contain the names of the roles.",
	},
	"authMethodConfig": {
		"Register an authMethodConfig with the backend.",
		`A authMethodConfig is required to authenticate with this backend. The authMethodConfig binds
		JWT token information with token policies and settings.
		The bindings, token polices and token settings can all be configured
		using this endpoint`,
	},
}
