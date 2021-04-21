package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/tokenutil"
	"github.com/hashicorp/vault/sdk/logical"
	"gopkg.in/square/go-jose.v2/jwt"
)

var reservedMetadata = []string{"authMethodConfig"}

const (
	claimDefaultLeeway    = 150
	boundClaimsTypeString = "string"
	boundClaimsTypeGlob   = "glob"
)

const (
	methodTypeJWT        = "jwt"
	methodTypeOIDC       = "oidc"
	methodTypeOwn        = "jwt_own"
	methodTypeSAPassword = "service_account_password"
)

var authMethodTypes = []string{methodTypeOwn, methodTypeJWT, methodTypeOIDC, methodTypeSAPassword}

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
					methodTypeJWT, methodTypeOIDC, methodTypeSAPassword, methodTypeOwn),
				Required: true,
			},
			"source": {
				Type: framework.TypeString,
				Description: fmt.Sprintf("authentification source for method thypes '%s' and '%s'.",
					methodTypeJWT, methodTypeOIDC),
			},

			"expiration_leeway": {
				Type: framework.TypeSignedDurationSecond,
				Description: `Duration in seconds of leeway when validating expiration of a token to account for clock skew. 
Defaults to 150 (2.5 minutes) if set to 0 and can be disabled if set to -1.`,
				Default: claimDefaultLeeway,
			},
			"not_before_leeway": {
				Type: framework.TypeSignedDurationSecond,
				Description: `Duration in seconds of leeway when validating not before values of a token to account for clock skew. 
Defaults to 150 (2.5 minutes) if set to 0 and can be disabled if set to -1.`,
				Default: claimDefaultLeeway,
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
				Default:     boundClaimsTypeString,
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

type authMethodConfig struct {
	tokenutil.TokenParams

	MethodType string `json:"method_type"`

	Source string `json:"source"`

	// Duration of leeway for expiration to account for clock skew
	ExpirationLeeway time.Duration `json:"expiration_leeway"`

	// Duration of leeway for not before to account for clock skew
	NotBeforeLeeway time.Duration `json:"not_before_leeway"`

	// Duration of leeway for all claims to account for clock skew
	ClockSkewLeeway time.Duration `json:"clock_skew_leeway"`

	// Role binding properties
	BoundAudiences      []string               `json:"bound_audiences"`
	BoundSubject        string                 `json:"bound_subject"`
	BoundClaimsType     string                 `json:"bound_claims_type"`
	BoundClaims         map[string]interface{} `json:"bound_claims"`
	ClaimMappings       map[string]string      `json:"claim_mappings"`
	UserClaim           string                 `json:"user_claim"`
	GroupsClaim         string                 `json:"groups_claim"`
	OIDCScopes          []string               `json:"oidc_scopes"`
	AllowedRedirectURIs []string               `json:"allowed_redirect_uris"`
	VerboseOIDCLogging  bool                   `json:"verbose_oidc_logging"`
	MaxAge              time.Duration          `json:"max_age"`
}

// authMethodConfig takes a storage backend and the name and returns the authMethodConfig's storage
// entry
func (b *flantIamAuthBackend) authMethod(ctx context.Context, s logical.Storage, name string) (*authMethodConfig, error) {
	raw, err := s.Get(ctx, rolePrefix+name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	role := new(authMethodConfig)
	if err := raw.DecodeJSON(role); err != nil {
		return nil, err
	}

	// Report legacy roles as type "jwt"
	if role.MethodType == "" {
		role.MethodType = "jwt"
	}

	if role.BoundClaimsType == "" {
		role.BoundClaimsType = boundClaimsTypeString
	}

	return role, nil
}

// pathAuthMethodExistenceCheck returns whether the authMethodConfig with the given name exists or not.
func (b *flantIamAuthBackend) pathAuthMethodExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	methodName, err := b.authMethod(ctx, req.Storage, data.Get("name").(string))
	if err != nil {
		return false, err
	}
	return methodName != nil, nil
}

// pathRoleList is used to list all the Roles registered with the backend.
func (b *flantIamAuthBackend) pathRoleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methods, err := req.Storage.List(ctx, rolePrefix)
	if err != nil {
		return nil, err
	}
	return logical.ListResponse(methods), nil
}

// pathAuthMethodRead grabs a read lock and reads the options set on the authMethodConfig from the storage
func (b *flantIamAuthBackend) pathAuthMethodRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	methodName := data.Get("name").(string)
	if methodName == "" {
		return logical.ErrorResponse("missing name"), nil
	}

	method, err := b.authMethod(ctx, req.Storage, methodName)
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

	if method.MethodType == methodTypeOwn {
		d["expiration_leeway"] = int64(method.ExpirationLeeway.Seconds())
		d["not_before_leeway"] = int64(method.NotBeforeLeeway.Seconds())
		d["clock_skew_leeway"] = int64(method.ClockSkewLeeway.Seconds())
	}

	if method.MethodType == methodTypeOwn || method.MethodType == methodTypeSAPassword {
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

	// Delete the authMethodConfig itself
	if err := req.Storage.Delete(ctx, rolePrefix+methodName); err != nil {
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

	// Check if the auth already exists
	method, err := b.authMethod(ctx, req.Storage, methodName)
	if err != nil {
		return nil, err
	}

	// Create a new entry object if this is a CreateOperation
	if method == nil {
		if req.Operation == logical.UpdateOperation {
			return nil, errors.New("auth method entry not found during update operation")
		}
		method = new(authMethodConfig)
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

	if methodType == methodTypeJWT || methodType == methodTypeOIDC {
		sourceName := data.Get("source").(string)
		if sourceName == "" {
			return logical.ErrorResponse("missing source"), nil
		}

		source, err := b.authSource(ctx, req, sourceName)
		if err != nil {
			return nil, err
		}

		if source == nil {
			return logical.ErrorResponse(fmt.Sprintf("'%s': auth source not found", sourceName)), nil
		}

		authType := source.authType()
		if methodType == methodTypeJWT && !(authType == StaticKeys || authType == JWKS) {
			return logical.ErrorResponse(fmt.Sprintf("incorrect source '%s': need jwt based source", sourceName)), nil
		}

		if methodType == methodTypeOIDC && !(authType == OIDCFlow || authType == OIDCDiscovery) {
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

	if methodType == methodTypeOwn {
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

	if !(methodType == methodTypeOwn || methodType == methodTypeSAPassword) {
		method.TokenParams.TokenTTL = 0
		method.TokenParams.TokenMaxTTL = 0
		method.TokenParams.TokenPolicies = []string{}
		method.TokenParams.TokenBoundCIDRs = []*sockaddr.SockAddrMarshaler{}
		method.TokenParams.TokenExplicitMaxTTL = 0
		method.TokenParams.TokenNoDefaultPolicy = false
		method.TokenParams.TokenNumUses = 0
		method.TokenParams.TokenPeriod = 0
		method.TokenParams.TokenType = logical.TokenTypeDefault
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
	case boundClaimsTypeString, boundClaimsTypeGlob:
		method.BoundClaimsType = boundClaimsType
	default:
		return logical.ErrorResponse("invalid 'bound_claims_type': %s", boundClaimsType), nil
	}

	if boundClaimsRaw, ok := data.GetOk("bound_claims"); ok {
		method.BoundClaims = boundClaimsRaw.(map[string]interface{})

		if boundClaimsType == boundClaimsTypeGlob {
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

	if method.MethodType == methodTypeOIDC && len(method.AllowedRedirectURIs) == 0 {
		return logical.ErrorResponse(
			"'allowed_redirect_uris' must be set if 'method_type' is 'oidc' or unspecified."), nil
	}

	// OIDC verification will enforce that the audience match the configured client_id.
	// For other methods, require at least one bound constraint.
	if methodType == methodTypeJWT {
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

	// Store the entry.
	entry, err := logical.StorageEntryJSON(rolePrefix+methodName, method)
	if err != nil {
		return nil, err
	}
	if err = req.Storage.Put(ctx, entry); err != nil {
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
