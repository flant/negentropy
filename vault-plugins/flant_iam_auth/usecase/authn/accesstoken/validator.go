// Package accesstoken  is designed on the base of github.com/hashicorp/cap/jwt.
// Some parts are identical, others are different
package accesstoken

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"time"

	hcjwt "github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/go-cleanhttp"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	jwt2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/jwt"
)

// DefaultLeewaySeconds defines the amount of leeway that's used by default
// for validating the "nbf" (Not Before) and "exp" (Expiration Time) claims.
const DefaultLeewaySeconds = 150

type accessTokenValidator struct {
	userinfoURL string
	issuer      string
	client      *http.Client
}

func NewAccessTokenValidator(ctx context.Context, cfg *model.AuthSource) (jwt2.JwtValidator, error) {
	if cfg == nil || cfg.OIDCDiscoveryURL == "" {
		return nil, fmt.Errorf("%v", cfg)
	}
	caCtx, err := createCAContext(ctx, cfg.OIDCDiscoveryCAPEM)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient
	if c, ok := caCtx.Value(oauth2.HTTPClient).(*http.Client); ok {
		client = c
	}

	wellKnown := strings.TrimSuffix(cfg.OIDCDiscoveryURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequest(http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req.WithContext(caCtx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body and status code
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	// Unmarshal the response body to obtain the issuer and JWKS URL
	var p struct {
		Issuer           string `json:"issuer"`
		UserInfoEndpoint string `json:"userinfo_endpoint"`
	}
	if err := unmarshalResp(resp, body, &p); err != nil {
		return nil, fmt.Errorf("failed to decode OIDC discovery document: %w", err)
	}

	return &accessTokenValidator{
		userinfoURL: p.UserInfoEndpoint,
		issuer:      p.Issuer,
		client:      client,
	}, nil
}

// Validate validates access_token JWT
//
// The given JWT is considered valid if:
//  0. iss match issuer at OIDCDiscoveryURL
//  1. oidc-provider returns userinfo presenting access_token
//  2. Its claims set, header parameter and userinfo response values match what's given by Expected.
//  3. It's valid with respect to the current time. This means that the current
//     time must be within the times (inclusive) given by the "nbf" (Not Before)
//     and "exp" (Expiration Time) claims and after the time given by the "iat"
//     (Issued At) claim, with configurable leeway. See Expected.Now() for details
//     on how the current time is provided for validation.
func (v *accessTokenValidator) Validate(ctx context.Context, token string, expected hcjwt.Expected) (map[string]interface{}, error) {
	// First, verify the signature to ensure subsequent validation is against verified claims
	allClaims, err := v.enrichedClaims(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("error verifying token: %w", err)
	}

	// Unmarshal all claims into the set of public JWT registered claims
	claims := jwt.Claims{}
	allClaimsJSON, err := json.Marshal(allClaims)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(allClaimsJSON, &claims); err != nil {
		return nil, err
	}

	// At least one of the "nbf" (Not Before), "exp" (Expiration Time), or "iat" (Issued At)
	// claims are required to be set.
	if claims.IssuedAt == nil {
		claims.IssuedAt = new(jwt.NumericDate)
	}
	if claims.Expiry == nil {
		claims.Expiry = new(jwt.NumericDate)
	}
	if claims.NotBefore == nil {
		claims.NotBefore = new(jwt.NumericDate)
	}
	if *claims.IssuedAt == 0 && *claims.Expiry == 0 && *claims.NotBefore == 0 {
		return nil, errors.New("no issued at (iat), not before (nbf), or expiration time (exp) claims in token")
	}

	// If "exp" (Expiration Time) is not set, then set it to the latest of
	// either the "iat" (Issued At) or "nbf" (Not Before) claims plus leeway.
	if *claims.Expiry == 0 {
		latestStart := *claims.IssuedAt
		if *claims.NotBefore > *claims.IssuedAt {
			latestStart = *claims.NotBefore
		}
		leeway := expected.ExpirationLeeway.Seconds()
		if expected.ExpirationLeeway.Seconds() < 0 {
			leeway = 0
		} else if expected.ExpirationLeeway.Seconds() == 0 {
			leeway = DefaultLeewaySeconds
		}
		*claims.Expiry = jwt.NumericDate(int64(latestStart) + int64(leeway))
	}

	// If "nbf" (Not Before) is not set, then set it to the "iat" (Issued At) if set.
	// Otherwise, set it to the "exp" (Expiration Time) minus leeway.
	if *claims.NotBefore == 0 {
		if *claims.IssuedAt != 0 {
			*claims.NotBefore = *claims.IssuedAt
		} else {
			leeway := expected.NotBeforeLeeway.Seconds()
			if expected.NotBeforeLeeway.Seconds() < 0 {
				leeway = 0
			} else if expected.NotBeforeLeeway.Seconds() == 0 {
				leeway = DefaultLeewaySeconds
			}
			*claims.NotBefore = jwt.NumericDate(int64(*claims.Expiry) - int64(leeway))
		}
	}

	// Set clock skew leeway to apply when validating all time-related claims
	cksLeeway := expected.ClockSkewLeeway
	if expected.ClockSkewLeeway.Seconds() < 0 {
		cksLeeway = 0
	} else if expected.ClockSkewLeeway.Seconds() == 0 {
		cksLeeway = jwt.DefaultLeeway
	}

	// Validate claims by asserting they're as expected
	if (expected.Issuer != "" && expected.Issuer != claims.Issuer) || v.issuer != claims.Issuer {
		return nil, fmt.Errorf("invalid issuer (iss) claim")
	}
	if expected.Subject != "" && expected.Subject != claims.Subject {
		return nil, fmt.Errorf("invalid subject (sub) claim")
	}
	if expected.ID != "" && expected.ID != claims.ID {
		return nil, fmt.Errorf("invalid ID (jti) claim")
	}
	if err := validateAudience(expected.Audiences, claims.Audience); err != nil {
		return nil, fmt.Errorf("invalid audience (aud) claim: %w", err)
	}

	// Validate that the token is not expired with respect to the current time
	now := time.Now()
	if expected.Now != nil {
		now = expected.Now()
	}
	if claims.NotBefore != nil && now.Add(cksLeeway).Before(claims.NotBefore.Time()) {
		return nil, errors.New("invalid not before (nbf) claim: token not yet valid")
	}
	if claims.Expiry != nil && now.Add(-cksLeeway).After(claims.Expiry.Time()) {
		return nil, errors.New("invalid expiration time (exp) claim: token is expired")
	}
	if claims.IssuedAt != nil && now.Add(cksLeeway).Before(claims.IssuedAt.Time()) {
		return nil, errors.New("invalid issued at (iat) claim: token issued in the future")
	}

	return allClaims, nil
}

// enrichedClaims validates access_token, and returns enriched claims
//
// The given JWT is considered valid if:
//  0. iss match issuer at OIDCDiscoveryURL
//  1. oidc-provider returns userinfo presenting access_token
//
//  enrichedClaims returns claims given by the token and new claims from userinfoURL
func (v *accessTokenValidator) enrichedClaims(ctx context.Context, token string) (map[string]interface{}, error) {
	userinfo, err := v.getUserInfo(ctx, token)

	parsedJWT, err := jwt.ParseSigned(token)
	if err != nil {
		return nil, err
	}
	allClaims := map[string]interface{}{}
	err = parsedJWT.UnsafeClaimsWithoutVerification(&allClaims)
	if err != nil {
		return nil, err
	}
	for k, v := range userinfo {
		if _, allredyExists := allClaims[k]; !allredyExists {
			allClaims[k] = v
		}
	}
	return allClaims, nil
}

func (v *accessTokenValidator) getUserInfo(ctx context.Context, token string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, v.userinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := v.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	userinfo := map[string]interface{}{}
	if err := unmarshalResp(resp, body, &userinfo); err != nil {
		return nil, fmt.Errorf("failed to decode OIDC discovery document: %w", err)
	}
	return userinfo, nil
}

// validateAudience returns an error if audClaim does not contain any audiences
// given by expectedAudiences. If expectedAudiences is empty, it skips validation
// and returns nil.
func validateAudience(expectedAudiences, audClaim []string) error {
	if len(expectedAudiences) == 0 {
		return nil
	}

	for _, v := range expectedAudiences {
		if contains(audClaim, v) {
			return nil
		}
	}

	return errors.New("audience claim does not match any expected audience")
}

func contains(sl []string, st string) bool {
	for _, s := range sl {
		if s == st {
			return true
		}
	}
	return false
}

// createCAContext returns a context with a custom TLS client that's configured with the root
// certificates from caPEM. If no certificates are configured, the original context is returned.
func createCAContext(ctx context.Context, caPEM string) (context.Context, error) {
	if caPEM == "" {
		return ctx, nil
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(caPEM)); !ok {
		return nil, errors.New("could not parse CA PEM value successfully")
	}

	tr := cleanhttp.DefaultPooledTransport()
	tr.TLSClientConfig = &tls.Config{
		RootCAs: certPool,
	}
	tc := &http.Client{
		Transport: tr,
	}

	caCtx := context.WithValue(ctx, oauth2.HTTPClient, tc)

	return caCtx, nil
}

// unmarshalResp JSON unmarshals the given body into the value pointed to by v.
// If it is unable to JSON unmarshal body into v, then it returns an appropriate
// error based on the Content-Type header of r.
func unmarshalResp(r *http.Response, body []byte, v interface{}) error {
	err := json.Unmarshal(body, &v)
	if err == nil {
		return nil
	}
	ct := r.Header.Get("Content-Type")
	mediaType, _, parseErr := mime.ParseMediaType(ct)
	if parseErr == nil && mediaType == "application/json" {
		return fmt.Errorf("got Content-Type = application/json, but could not unmarshal as JSON: %v", err)
	}
	return fmt.Errorf("expected Content-Type = application/json, got %q: %v", ct, err)
}
