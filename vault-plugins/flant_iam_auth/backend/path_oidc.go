package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/jwt"
)

const (
	oidcRequestTimeout         = 10 * time.Minute
	oidcRequestCleanupInterval = 1 * time.Minute
)

const (
	// OIDC error prefixes. These are searched for specifically by the UI, so any
	// changes to them must be aligned with a UI change.
	errLoginFailed       = "Vault login failed."
	errNoResponse        = "No response from provider."
	errTokenVerification = "Token verification failed."
	errNotOIDCFlow       = "OIDC login is not configured for this mount"

	noCode = "no_code"
)

// oidcRequest represents a single OIDC authentication flow. It is created when
// an authURL is requested. It is uniquely identified by a state, which is passed
// throughout the multiple interactions needed to complete the flow.
type oidcRequest struct {
	oidc.Request

	method  string
	code    string
	idToken string

	// clientNonce is used between Vault and the client/application (e.g. CLI) making the request,
	// and is unrelated to the OIDC nonce above. It is optional.
	clientNonce string
}

func pathOIDC(b *flantIamAuthBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: `oidc/callback`,
			Fields: map[string]*framework.FieldSchema{
				"state": {
					Type: framework.TypeString,
				},
				"code": {
					Type: framework.TypeString,
				},
				"id_token": {
					Type: framework.TypeString,
				},
				"client_nonce": {
					Type: framework.TypeString,
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathCallback,
					Summary:  "Callback endpoint to complete an OIDC login.",

					// state is cached so don't process OIDC logins on perf standbys
					ForwardPerformanceStandby: true,
				},

				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathCallbackPost,
					Summary:  "Callback endpoint to handle form_posts.",

					// state is cached so don't process OIDC logins on perf standbys
					ForwardPerformanceStandby: true,
				},
			},
		},
		{
			Pattern: `oidc/auth_url`,
			Fields: map[string]*framework.FieldSchema{
				"authMethodName": {
					Type:        framework.TypeLowerCaseString,
					Description: "The authMethodConfig to issue an OIDC authorization URL against.",
				},
				"redirect_uri": {
					Type:        framework.TypeString,
					Description: "The OAuth redirect_uri to use in the authorization URL.",
				},
				"client_nonce": {
					Type:        framework.TypeString,
					Description: "Optional client-provided nonce that must match during callback, if present.",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.authURL,
					Summary:  "Request an authorization URL to start an OIDC login flow.",

					// state is cached so don't process OIDC logins on perf standbys
					ForwardPerformanceStandby: true,
				},
			},
		},
	}
}

func (b *flantIamAuthBackend) pathCallbackPost(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	stateID := d.Get("state").(string)
	code := d.Get("code").(string)
	idToken := d.Get("id_token").(string)

	resp := &logical.Response{
		Data: map[string]interface{}{
			logical.HTTPContentType: "text/html",
			logical.HTTPStatusCode:  http.StatusOK,
		},
	}

	requestRaw, ok := b.oidcRequests.Get(stateID)
	if !ok {
		return nil, errors.New("OIDC state not found")
	}

	oidcReq := requestRaw.(*oidcRequest)

	// Ensure that the Role still exists.
	tnx := b.storage.Txn(false)
	repo := repo2.NewAuthMethodRepo(tnx)
	method, err := repo.Get(oidcReq.method)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return logical.ErrorResponse(errLoginFailed + " Could not load configuration."), nil
	}

	if method.MethodType != model.MethodTypeOIDC {
		return logical.ErrorResponse(errLoginFailed + " Incorrect method."), nil
	}

	authSource, err := repo2.NewAuthSourceRepo(tnx).Get(method.Source)
	if err != nil {
		return nil, err
	}
	if authSource == nil {
		return logical.ErrorResponse(errLoginFailed + " auth source for method could not be found"), nil
	}

	if authSource.OIDCResponseMode != model.ResponseModeFormPost {
		return logical.RespondWithStatusCode(nil, req, http.StatusMethodNotAllowed)
	}

	oidcReq, err = b.amendOIDCRequest(stateID, code, idToken)
	if err != nil {
		resp.Data[logical.HTTPRawBody] = []byte(errorHTML(errLoginFailed, "Expired or missing OAuth state."))
		resp.Data[logical.HTTPStatusCode] = http.StatusBadRequest
	} else {
		mount := parseMount(oidcReq.RedirectURL())
		if mount == "" {
			resp.Data[logical.HTTPRawBody] = []byte(errorHTML(errLoginFailed, "Invalid redirect path."))
			resp.Data[logical.HTTPStatusCode] = http.StatusBadRequest
		} else {
			resp.Data[logical.HTTPRawBody] = []byte(formpostHTML(mount, noCode, stateID))
		}
	}

	return resp, nil
}

func (b *flantIamAuthBackend) pathCallback(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	stateID := d.Get("state").(string)

	oidcReq := b.verifyOIDCRequest(stateID)
	if oidcReq == nil {
		return logical.ErrorResponse(errLoginFailed + " Expired or missing OAuth state."), nil
	}

	clientNonce := d.Get("client_nonce").(string)

	// If a client_nonce was provided at the start of the auth process as part of the auth_url
	// request, require that it is present and matching during the callback phase.
	if oidcReq.clientNonce != "" && clientNonce != oidcReq.clientNonce {
		return logical.ErrorResponse("invalid client_nonce"), nil
	}

	methodName := oidcReq.method

	tnx := b.storage.Txn(false)
	repo := repo2.NewAuthMethodRepo(tnx)
	method, err := repo.Get(oidcReq.method)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return logical.ErrorResponse(errLoginFailed + " authMethod could not be found"), nil
	}

	if method.MethodType != model.MethodTypeOIDC {
		return logical.ErrorResponse(errLoginFailed + " authMethod type must be OIDC"), nil
	}

	authSource, err := repo2.NewAuthSourceRepo(tnx).Get(method.Source)
	if err != nil {
		return nil, err
	}
	if authSource == nil {
		return logical.ErrorResponse(errLoginFailed + " auth source for method could not be found"), nil
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

	provider, err := b.getProvider(authSource)
	if err != nil {
		return nil, errwrap.Wrapf("error getting provider for login operation: {{err}}", err)
	}

	var rawToken oidc.IDToken
	var token *oidc.Tk

	code := d.Get("code").(string)
	if code == noCode {
		code = oidcReq.code
	}

	if code == "" {
		if oidcReq.idToken == "" {
			return logical.ErrorResponse(errLoginFailed + " No code or id_token received."), nil
		}

		// Verify the ID token received from the authentication response.
		rawToken = oidc.IDToken(oidcReq.idToken)
		if _, err := provider.VerifyIDToken(ctx, rawToken, oidcReq); err != nil {
			return logical.ErrorResponse("%s %s", errTokenVerification, err.Error()), nil
		}
	} else {
		// Exchange the authorization code for an ID token and access token.
		// ID token verification takes place in provider.Exchange.
		token, err = provider.Exchange(ctx, oidcReq, stateID, code)
		if err != nil {
			return logical.ErrorResponse(errLoginFailed+" Error exchanging oidc code: %q.", err.Error()), nil
		}

		rawToken = token.IDToken()
	}

	if method.VerboseOIDCLogging {
		loggedToken := "invalid token format"

		parts := strings.Split(string(rawToken), ".")
		if len(parts) == 3 {
			// strip signature from logged token
			loggedToken = fmt.Sprintf("%s.%s.xxxxxxxxxxx", parts[0], parts[1])
		}

		b.Logger().Debug("OIDC provider response", "id_token", loggedToken)
	}

	// Parse claims from the ID token payload.
	var allClaims map[string]interface{}
	if err := rawToken.Claims(&allClaims); err != nil {
		return nil, err
	}
	delete(allClaims, "nonce")

	// Get the subject claim for bound subject and user info validation
	var subject string
	if subStr, ok := allClaims["sub"].(string); ok {
		subject = subStr
	}

	if method.BoundSubject != "" && method.BoundSubject != subject {
		return nil, errors.New("sub claim does not match bound subject")
	}

	// Set the token source for the access token if it's available. It will only
	// be available for the authorization code flow (oidc_response_types=code).
	// The access token will be used for fetching additional user and group info.
	var tokenSource oauth2.TokenSource
	if token != nil {
		tokenSource = token.StaticTokenSource()
	}

	// If we have a token, attempt to fetch information from the /userinfo endpoint
	// and merge it with the existing claims data. A failure to fetch additional information
	// from this endpoint will not invalidate the authorization flow.
	if tokenSource != nil {
		if err := provider.UserInfo(ctx, tokenSource, subject, &allClaims); err != nil {
			logFunc := b.Logger().Warn
			if strings.Contains(err.Error(), "user info endpoint is not supported") {
				logFunc = b.Logger().Info
			}
			logFunc("error reading /userinfo endpoint", "error", err)
		}
	}

	if method.VerboseOIDCLogging {
		if c, err := json.Marshal(allClaims); err == nil {
			b.Logger().Debug("OIDC provider response", "claims", string(c))
		} else {
			b.Logger().Debug("OIDC provider response", "marshalling error", err.Error())
		}
	}

	alias, _, err := jwt.CreateIdentity(b.Logger(), allClaims, method, tokenSource)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	if err := jwt.ValidateBoundClaims(b.Logger(), method.BoundClaimsType, method.BoundClaims, allClaims); err != nil {
		return logical.ErrorResponse("error validating claims: %s", err.Error()), nil
	}

	tokenMetadata := map[string]string{"flantIamAuthMethod": methodName}
	for k, v := range alias.Metadata {
		tokenMetadata[k] = v
	}

	auth := &logical.Auth{
		Policies:    method.TokenPolicies,
		DisplayName: alias.Name,
		Period:      method.TokenPeriod,
		NumUses:     method.TokenNumUses,
		Alias:       alias,
		InternalData: map[string]interface{}{
			"flantIamAuthMethod": methodName,
		},
		Metadata: tokenMetadata,
		LeaseOptions: logical.LeaseOptions{
			Renewable: true,
			TTL:       method.TokenTTL,
			MaxTTL:    method.TokenMaxTTL,
		},
		BoundCIDRs: method.TokenBoundCIDRs,
	}

	method.PopulateTokenAuth(auth)

	resp := &logical.Response{
		Auth: auth,
	}

	return resp, nil
}

// authURL returns a URL used for redirection to receive an authorization code.
// This path requires a authMethodConfig name, or that a default_role has been configured.
// Because this endpoint is unauthenticated, the response to invalid or non-OIDC
// roles is intentionally non-descriptive and will simply be an empty string.
func (b *flantIamAuthBackend) authURL(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	logger := b.Logger()

	authMethodName := d.Get("authMethodName").(string)

	if authMethodName == "" {
		return logical.ErrorResponse("missing authMethodName"), nil
	}

	redirectURI := d.Get("redirect_uri").(string)
	if redirectURI == "" {
		return logical.ErrorResponse("missing redirect_uri"), nil
	}

	clientNonce := d.Get("client_nonce").(string)

	tnx := b.storage.Txn(false)
	repo := repo2.NewAuthMethodRepo(tnx)
	method, err := repo.Get(authMethodName)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return logical.ErrorResponse("authMethodName %q could not be found", authMethodName), nil
	}

	if method.MethodType != model.MethodTypeOIDC {
		return logical.ErrorResponse("authMethodConfig %q could not be found", authMethodName), nil
	}

	// default response for most error/invalid conditions
	resp := &logical.Response{
		Data: map[string]interface{}{
			"auth_url": "",
		},
	}

	authSource, err := repo2.NewAuthSourceRepo(tnx).Get(method.Source)
	if err != nil {
		return nil, err
	}
	if authSource == nil {
		return logical.ErrorResponse("could not load configuration"), nil
	}

	if authSource.AuthType() != model.AuthSourceOIDCFlow {
		return logical.ErrorResponse(errNotOIDCFlow), nil
	}

	// If namespace will be passed around in state, and it has been provided as
	// a redirectURI query parameter, remove it from redirectURI, and append it
	// to the state (later in this function)
	namespace := ""
	if authSource.NamespaceInState {
		inputURI, err := url.Parse(redirectURI)
		if err != nil {
			return resp, nil
		}
		qParam := inputURI.Query()
		namespace = qParam.Get("namespace")
		if len(namespace) > 0 {
			qParam.Del("namespace")
			inputURI.RawQuery = qParam.Encode()
			redirectURI = inputURI.String()
		}
	}

	if !validRedirect(redirectURI, method.AllowedRedirectURIs) {
		logger.Warn("unauthorized redirect_uri", "redirect_uri", redirectURI)
		return resp, nil
	}

	provider, err := b.getProvider(authSource)
	if err != nil {
		logger.Warn("error getting provider for login operation", "error", err)
		return resp, nil
	}

	oidcReq, err := b.createOIDCRequest(authSource, method, authMethodName, redirectURI, clientNonce)
	if err != nil {
		logger.Warn("error generating OAuth state", "error", err)
		return resp, nil
	}

	urlStr, err := provider.AuthURL(ctx, oidcReq)
	if err != nil {
		logger.Warn("error generating auth URL", "error", err)
		return resp, nil
	}

	// embed namespace in state in the auth_url
	if authSource.NamespaceInState && len(namespace) > 0 {
		stateWithNamespace := fmt.Sprintf("%s,ns=%s", oidcReq.State(), namespace)
		urlStr = strings.Replace(urlStr, oidcReq.State(), url.QueryEscape(stateWithNamespace), 1)
	}

	resp.Data["auth_url"] = urlStr

	return resp, nil
}

// createOIDCRequest makes an expiring request object, associated with a random state ID
// that is passed throughout the OAuth process. A nonce is also included in the auth process.
func (b *flantIamAuthBackend) createOIDCRequest(config *model.AuthSource, method *model.AuthMethod, methodNAme, redirectURI, clientNonce string) (*oidcRequest, error) {
	options := []oidc.Option{
		oidc.WithAudiences(method.BoundAudiences...),
		oidc.WithScopes(method.OIDCScopes...),
	}

	if config.HasType(model.ResponseTypeIDToken) {
		options = append(options, oidc.WithImplicitFlow())
	}

	if method.MaxAge > 0 {
		options = append(options, oidc.WithMaxAge(uint(method.MaxAge.Seconds())))
	}

	request, err := oidc.NewRequest(oidcRequestTimeout, redirectURI, options...)
	if err != nil {
		return nil, err
	}

	oidcReq := &oidcRequest{
		Request:     request,
		method:      methodNAme,
		clientNonce: clientNonce,
	}
	b.oidcRequests.SetDefault(request.State(), oidcReq)

	return oidcReq, nil
}

func (b *flantIamAuthBackend) amendOIDCRequest(stateID, code, idToken string) (*oidcRequest, error) {
	requestRaw, ok := b.oidcRequests.Get(stateID)
	if !ok {
		return nil, errors.New("OIDC state not found")
	}

	oidcReq := requestRaw.(*oidcRequest)
	oidcReq.code = code
	oidcReq.idToken = idToken

	b.oidcRequests.SetDefault(stateID, oidcReq)

	return oidcReq, nil
}

// verifyOIDCRequest tests whether the provided state ID is valid and returns the
// associated oidcRequest if so. A nil oidcRequest is returned if the ID is not found
// or expired. The oidcRequest should only ever be retrieved once and is deleted as
// part of this request.
func (b *flantIamAuthBackend) verifyOIDCRequest(stateID string) *oidcRequest {
	defer b.oidcRequests.Delete(stateID)

	if requestRaw, ok := b.oidcRequests.Get(stateID); ok {
		return requestRaw.(*oidcRequest)
	}

	return nil
}

// validRedirect checks whether uri is in allowed using special handling for loopback uris.
// Ref: https://tools.ietf.org/html/rfc8252#section-7.3
func validRedirect(uri string, allowed []string) bool {
	inputURI, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// if uri isn't a loopback, just string search the allowed list
	if !strutil.StrListContains([]string{"localhost", "127.0.0.1", "::1"}, inputURI.Hostname()) {
		return strutil.StrListContains(allowed, uri)
	}

	// otherwise, search for a match in a port-agnostic manner, per the OAuth RFC.
	inputURI.Host = inputURI.Hostname()

	for _, a := range allowed {
		allowedURI, err := url.Parse(a)
		if err != nil {
			return false
		}
		allowedURI.Host = allowedURI.Hostname()

		if inputURI.String() == allowedURI.String() {
			return true
		}
	}

	return false
}

// parseMount attempts to extract the mount path from a redirect URI.
func parseMount(redirectURI string) string {
	parts := strings.Split(redirectURI, "/")

	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "v1" && parts[i+1] == "auth" {
			return parts[i+2]
		}
	}
	return ""
}
