package jwtauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/hashicorp/vault/sdk/helper/parseutil"
	"github.com/hashicorp/vault/sdk/helper/tokenutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
)

type errCase struct {
	title         string
	body          map[string]interface{}
	errPrefix     string
	hasBackendErr bool
}

func assertAuthMethod(t *testing.T, b *flantIamAuthBackend, methodName string, expected model.AuthMethod) {
	actual, err := repo.NewAuthMethodRepo(b.storage.Txn(false)).Get(methodName)
	if err != nil {
		t.Fatal(err)
	}

	if actual.UUID == "" {
		t.Fatal("not set uuid")
	}

	uuid := actual.UUID
	defer func() {
		actual.UUID = uuid
	}()

	actual.UUID = ""

	if diff := deep.Equal(expected, *actual); diff != nil {
		t.Fatalf("Unexpected authMethod data: diff %#v\n", diff)
	}
}

func createJwtAuthMethod(t *testing.T, b *flantIamAuthBackend, storage logical.Storage, methodName, jwtSourceName string) {
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      fmt.Sprintf("auth_method/%s", methodName),
		Storage:   storage,
		Data: withBoundClaims(
			withLeaways(
				withVaultTokenParts(
					withUserClaims(map[string]interface{}{
						"method_type": model.MethodTypeJWT,
						"source":      jwtSourceName,
					})))),
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
}

func assertErrorCasesAuthMethod(t *testing.T, b logical.Backend, storage logical.Storage, cases []errCase) {
	for _, c := range cases {
		t.Run(fmt.Sprintf("does not create method %s", c.title), func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/err_method",
				Storage:   storage,
				Data:      c.body,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				if c.hasBackendErr {
					return
				}
				t.Fatalf("err:%s or response is nil", err)
			}

			if resp == nil || !resp.IsError() {
				t.Fatal("expected error")
			}

			if c.errPrefix != "" && !strings.Contains(resp.Error().Error(), c.errPrefix) {
				t.Fatalf("got unexpected error: %v, need %v", resp.Error(), c.errPrefix)
			}
		})
	}
}

func enableJwtBackend(t *testing.T, b logical.Backend, storage logical.Storage) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwt/enable",
		Storage:   storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if resp != nil && resp.IsError() || err != nil {
		t.Fatalf("error enable jwt %v %v", resp, err)
	}
}

func disableJwtBackend(t *testing.T, b logical.Backend, storage logical.Storage) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwt/disable",
		Storage:   storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if resp != nil && resp.IsError() || err != nil {
		t.Fatalf("error enable jwt %v %v", resp, err)
	}
}

func withVaultTokenParts(body map[string]interface{}) map[string]interface{} {
	tokenPart := map[string]interface{}{
		"token_bound_cidrs":       []string{"127.0.0.1/8"},
		"token_explicit_max_ttl":  "100s",
		"token_max_ttl":           "100s",
		"token_no_default_policy": true,
		"token_period":            "10s",
		"token_policies":          []string{"good"},
		"token_type":              "default",
		"token_ttl":               "5s",
		"token_num_uses":          5,
	}

	for k, v := range tokenPart {
		body[k] = v
	}

	return body
}

func expectedWithTokenParams(m model.AuthMethod) model.AuthMethod {
	cidrsObj, err := parseutil.ParseAddrs([]string{"127.0.0.1/8"})
	if err != nil {
		panic(err)
	}

	m.TokenParams = tokenutil.TokenParams{
		TokenType:            logical.TokenTypeDefault,
		TokenTTL:             5 * time.Second,
		TokenMaxTTL:          100 * time.Second,
		TokenNumUses:         5,
		TokenPeriod:          10 * time.Second,
		TokenExplicitMaxTTL:  100 * time.Second,
		TokenPolicies:        []string{"good"},
		TokenNoDefaultPolicy: true,
		TokenBoundCIDRs:      cidrsObj,
	}

	return m
}

func withLeaways(body map[string]interface{}) map[string]interface{} {
	body["expiration_leeway"] = "5s"
	body["not_before_leeway"] = "5s"
	body["clock_skew_leeway"] = "5s"

	return body
}

func expectedWithLeaways(m model.AuthMethod) model.AuthMethod {
	m.ExpirationLeeway = 5 * time.Second
	m.NotBeforeLeeway = 5 * time.Second
	m.ClockSkewLeeway = 5 * time.Second

	return m
}

func withBoundClaims(body map[string]interface{}) map[string]interface{} {
	body["bound_subject"] = "testsub"
	body["bound_audiences"] = "vault"
	body["bound_claims_type"] = "glob"
	body["bound_claims"] = map[string]interface{}{
		"foo": []interface{}{"baz"},
	}
	body["bound_cidrs"] = "127.0.0.1/8"

	return body
}

func expectedWithBoundClaims(m model.AuthMethod) model.AuthMethod {
	m.BoundSubject = "testsub"
	m.BoundAudiences = []string{"vault"}
	m.BoundClaimsType = "glob"
	m.BoundClaims = map[string]interface{}{
		"foo": []interface{}{"baz"},
	}

	return m
}

func withUserClaims(body map[string]interface{}) map[string]interface{} {
	body["user_claim"] = "user"
	body["groups_claim"] = "groups"
	body["claim_mappings"] = map[string]string{
		"foo": "a",
	}

	return body
}

func expectedWithUser(m model.AuthMethod) model.AuthMethod {
	m.UserClaim = "user"
	m.GroupsClaim = "groups"
	m.ClaimMappings = map[string]string{
		"foo": "a",
	}

	return m
}

func TestAuthMethod_CreateError(t *testing.T) {
	t.Run("incorrect method type", func(t *testing.T) {
		b, storage := getBackend(t)
		jwtSourceName := "a"
		oidcSourceName := "b"

		enableJwtBackend(t, b, storage)
		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "not pass method type",
				body: map[string]interface{}{
					"bound_subject":     "testsub",
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
				},
			},

			{
				title: "incorrect method type",
				body: map[string]interface{}{
					"method_type":       "incorrect",
					"bound_claims_type": "string",
					"bound_subject":     "testsub",
					"bound_audiences":   "vault",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
				},
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)

		disableJwtBackend(t, b, storage)
		casesDisable := []errCase{
			{
				title: " multipass method does not create if jwt disabled",
				body: map[string]interface{}{
					"method_type": model.MethodTypeMultipass,
					"source":      jwtSourceName,
				},
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, casesDisable)
	})

	t.Run("incorrect relation between type and source", func(t *testing.T) {
		b, storage := getBackend(t)
		jwtSourceName := "a"
		oidcSourceName := "b"

		enableJwtBackend(t, b, storage)
		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "jwt type need source",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"bound_subject":     "testsub",
					"bound_claims_type": "string",
					"bound_audiences":   "vault",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
				},
			},

			{
				title: "oidc type need source",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"bound_claims": map[string]interface{}{
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "b",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
				},
			},

			{
				title: "multipass type don't need source",
				body: map[string]interface{}{
					"method_type": model.MethodTypeMultipass,
					"source":      jwtSourceName,
				},
			},

			{
				title: "sa password type don't need source",
				body: map[string]interface{}{
					"method_type": model.MethodTypeSAPassword,
					"source":      oidcSourceName,
				},
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("incorrect source", func(t *testing.T) {
		b, storage := getBackend(t)

		jwtSourceName := "a"
		oidcSourceName := "b"

		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "not found source",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"bound_subject":     "testsub",
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
					"source":            "not exists",
				},
			},

			{
				title: "jwt type with oidc source",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"bound_subject":     "testsub",
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
					"source":            oidcSourceName,
				},
			},

			{
				title: "oidc type with jwt source",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "b",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       jwtSourceName,
				},
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("incorrect bound claims", func(t *testing.T) {
		b, storage := getBackend(t)

		jwtSourceName := "a"
		oidcSourceName := "b"

		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "bound claims type for jwt",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"bound_claims_type": "invalid",
					"bound_subject":     "testsub",
					"bound_audiences":   "vault",
					"user_claim":        "user",
					"groups_claim":      "groups",
					"bound_cidrs":       "127.0.0.1/8",
					"source":            jwtSourceName,
				},

				errPrefix: "invalid 'bound_claims_type'",
			},

			{
				title: "obound claims type for oidc",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_claims_type": "invalid",
					"bound_audiences":   "vault",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "b",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "invalid 'bound_claims_type'",
			},

			{
				title: "bound claims glob for jwt",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"user_claim":        "user",
					"policies":          "test",
					"bound_claims_type": "glob",
					"bound_claims": map[string]interface{}{
						"bar": "baz",
						"foo": 25,
					},

					"source": jwtSourceName,
				},

				errPrefix: "claim is not a string or list",
			},

			{
				title: "bound claims glob for oidc",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_claims_type": "glob",
					"bound_claims": map[string]interface{}{
						"bar": "baz",
						"foo": 25,
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "b",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "claim is not a string or list",
			},

			{
				title: "bound claim glob is not string for jwt",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"user_claim":        "user",
					"policies":          "test",
					"bound_claims_type": "glob",
					"bound_claims": map[string]interface{}{
						"foo": []interface{}{"baz", 10},
					},

					"source": jwtSourceName,
				},

				errPrefix: "claim is not a string",
			},

			{
				title: "bound claim glob is not string for oidc",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_claims_type": "glob",
					"bound_claims": map[string]interface{}{
						"foo": []interface{}{"baz", 10},
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "b",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "claim is not a string",
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("oidc params", func(t *testing.T) {
		b, storage := getBackend(t)

		oidcSourceName := "b"

		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "allowed_redirect_uris is not passed",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"bound_claims": map[string]interface{}{
						"bar": "baz",
					},
					"oidc_scopes": []string{"email", "profile"},
					"claim_mappings": map[string]string{
						"foo": "a",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "allowed_redirect_uris' must be set if 'method_type' is 'oidc' or unspecified.",
			},

			{
				title: "max_age is negative",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeOIDC,
					"bound_audiences":   "vault",
					"bound_claims_type": "string",
					"bound_claims": map[string]interface{}{
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "a",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
					"max_age":      "-1s",
				},

				hasBackendErr: true,
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("user claims", func(t *testing.T) {
		b, storage := getBackend(t)

		jwtSourceName := "a"
		oidcSourceName := "b"

		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "metadata key in claims mapping for jwt",
				body: map[string]interface{}{
					"method_type":       model.MethodTypeJWT,
					"bound_claims_type": "string",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"bound_subject":   "testsub",
					"bound_audiences": "vault",
					"user_claim":      "user",
					"claim_mappings": map[string]string{
						"foo":        "a",
						"some_claim": "flantIamAuthMethod",
					},
					"groups_claim": "groups",
					"bound_cidrs":  "127.0.0.1/8",
					"source":       jwtSourceName,
				},

				errPrefix: "metadata key \"flantIamAuthMethod\" is reserved",
			},

			{
				title: "metadata key in claims mapping for oidc",
				body: map[string]interface{}{
					"method_type":     model.MethodTypeOIDC,
					"bound_audiences": "vault",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo":        "a",
						"some_claim": "flantIamAuthMethod",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "metadata key \"flantIamAuthMethod\" is reserved",
			},

			{
				title: "duplicate key destination in claims mapping for jwt",
				body: map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"bound_subject":   "testsub",
					"bound_audiences": "vault",
					"user_claim":      "user",
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "a",
					},
					"groups_claim": "groups",
					"bound_cidrs":  "127.0.0.1/8",
					"source":       jwtSourceName,
				},

				errPrefix: "multiple keys are mapped to metadata key",
			},

			{
				title: "duplicate key destination for oidc",
				body: map[string]interface{}{
					"method_type":     model.MethodTypeOIDC,
					"bound_audiences": "vault",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "a",
					},
					"user_claim":   "user",
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "multiple keys are mapped to metadata key",
			},

			{
				title: "must define user claim for jwt",
				body: map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"bound_subject":   "testsub",
					"bound_audiences": "vault",
					"claim_mappings": map[string]string{
						"foo": "a",
					},
					"groups_claim": "groups",
					"bound_cidrs":  "127.0.0.1/8",
					"source":       jwtSourceName,
				},

				errPrefix: "a user claim must be defined on the authMethod",
			},

			{
				title: "must define user claim for oidc",
				body: map[string]interface{}{
					"method_type":     model.MethodTypeOIDC,
					"bound_audiences": "vault",
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"oidc_scopes":           []string{"email", "profile"},
					"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
					"claim_mappings": map[string]string{
						"foo": "a",
					},
					"groups_claim": "groups",
					"source":       oidcSourceName,
				},

				errPrefix: "a user claim must be defined on the authMethod",
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("bound constraint", func(t *testing.T) {
		b, storage := getBackend(t)

		jwtSourceName := "a"
		oidcSourceName := "b"

		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		cases := []errCase{
			{
				title: "must one bound constraint for jwt",
				body: map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"user_claim":  "user",
					"claim_mappings": map[string]string{
						"foo":        "a",
						"some_claim": "flantIamAuthMethod",
					},
					"groups_claim": "groups",
					"source":       jwtSourceName,
				},

				errPrefix: "must have at least one bound constraint when creating/updating a authMethod",
			},
		}

		assertErrorCasesAuthMethod(t, b, storage, cases)
	})

	t.Run("vault token params", func(t *testing.T) {
		b, storage := getBackend(t)

		jwtSourceName := "a"
		oidcSourceName := "b"

		enableJwtBackend(t, b, storage)
		creteTestJWTBasedSource(t, b, storage, jwtSourceName)
		creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

		clone := func(o map[string]interface{}) map[string]interface{} {
			s, err := json.Marshal(o)
			if err != nil {
				t.Fatalf("error wile clone %v", err)
			}

			c := map[string]interface{}{}
			err = json.Unmarshal(s, &c)
			if err != nil {
				t.Fatalf("error wile clone %v", err)
			}

			return c
		}

		methods := []struct {
			body       map[string]interface{}
			methodName string
		}{
			{
				methodName: model.MethodTypeJWT,
				body: map[string]interface{}{
					"method_type":     model.MethodTypeJWT,
					"bound_subject":   "testsub",
					"bound_audiences": "vault",
					"user_claim":      "user",
					"groups_claim":    "groups",
					"bound_cidrs":     "127.0.0.1/8",
					"source":          jwtSourceName,
				},
			},

			{
				methodName: model.MethodTypeMultipass,
				body: map[string]interface{}{
					"method_type": model.MethodTypeMultipass,
				},
			},

			{
				methodName: model.MethodTypeSAPassword,
				body: map[string]interface{}{
					"method_type": model.MethodTypeSAPassword,
				},
			},

			{
				methodName: model.MethodTypeOIDC,
				body: map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"bound_claims": map[string]interface{}{
						"foo": 10,
						"bar": "baz",
					},
					"bound_subject":   "testsub",
					"bound_audiences": "vault",
					"user_claim":      "user",
					"claim_mappings": map[string]string{
						"foo": "a",
						"bar": "a",
					},
					"groups_claim": "groups",
					"bound_cidrs":  "127.0.0.1/8",
					"source":       jwtSourceName,
				},
			},
		}

		cases := []struct {
			title     string
			tokenPart map[string]interface{}
			errPrefix string
			hasErr    bool
		}{
			{
				title: "incorrect bound cidrs",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"invalid"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "100s",
					"token_no_default_policy": false,
					"token_period":            "10s",
					"token_policies":          []string{"good"},
					"token_type":              "default",
					"token_ttl":               "5s",
					"token_num_uses":          5,
				},
			},

			{
				title: "negative token max ttl",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "-1s",
					"token_no_default_policy": false,
					"token_period":            "10s",
					"token_policies":          []string{"good"},
					"token_type":              "default",
					"token_ttl":               "5s",
					"token_num_uses":          5,
				},

				hasErr: true,
			},

			{
				title: "incorrect token type",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "100s",
					"token_no_default_policy": false,
					"token_period":            "10s",
					"token_policies":          []string{"good"},
					"token_type":              "invalid",
					"token_ttl":               "5s",
					"token_num_uses":          5,
				},

				errPrefix: "invalid 'token_type' value",
			},

			{
				title: "cannot batch token with period",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "100s",
					"token_no_default_policy": false,
					"token_period":            "10s",
					"token_policies":          []string{"good"},
					"token_type":              "batch",
					"token_ttl":               "5s",
					"token_num_uses":          0,
				},

				errPrefix: "'token_type' cannot be 'batch' or 'default_batch' when set to generate periodic tokens",
			},

			{
				title: "cannot batch token with num uses",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "100s",
					"token_no_default_policy": false,
					"token_period":            "0s",
					"token_policies":          []string{"good"},
					"token_type":              "batch",
					"token_ttl":               "5s",
					"token_num_uses":          5,
				},

				errPrefix: "'token_type' cannot be 'batch' or 'default_batch' when set to generate tokens with limited use count",
			},

			{
				title: "negative token num uses",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "100s",
					"token_no_default_policy": false,
					"token_period":            "0s",
					"token_policies":          []string{"good"},
					"token_type":              "batch",
					"token_ttl":               "5s",
					"token_num_uses":          -5,
				},

				hasErr: true,
			},

			{
				title: "token max ttl less than token ttl",
				tokenPart: map[string]interface{}{
					"token_bound_cidrs":       []string{"127.0.0.1/8"},
					"token_explicit_max_ttl":  "100s",
					"token_max_ttl":           "1s",
					"token_no_default_policy": false,
					"token_period":            "0s",
					"token_policies":          []string{"good"},
					"token_type":              "batch",
					"token_ttl":               "500s",
					"token_num_uses":          5,
				},

				hasErr: true,
			},
		}

		allCasesPerMethods := make([]errCase, 0)

		for _, m := range methods {
			for _, c := range cases {
				body := clone(m.body)
				for k, v := range c.tokenPart {
					body[k] = v
				}

				allCasesPerMethods = append(allCasesPerMethods, errCase{
					title:         fmt.Sprintf("%s for %s", c.title, m.methodName),
					body:          body,
					errPrefix:     c.errPrefix,
					hasBackendErr: c.hasErr,
				})
			}
		}

		assertErrorCasesAuthMethod(t, b, storage, allCasesPerMethods)
	})
}

func TestAuthMethod_CreateUpdate(t *testing.T) {
	b, storage := getBackend(t)

	jwtSourceName := "a"
	oidcSourceName := "b"

	enableJwtBackend(t, b, storage)
	creteTestJWTBasedSource(t, b, storage, jwtSourceName)
	creteTestOIDCBasedSource(t, b, storage, oidcSourceName)

	methods := []struct {
		methodName string

		body     map[string]interface{}
		expected model.AuthMethod

		updateBody     map[string]interface{}
		updateExpected model.AuthMethod
	}{
		{
			methodName: model.MethodTypeJWT,
			body: withBoundClaims(
				withLeaways(
					withVaultTokenParts(
						withUserClaims(map[string]interface{}{
							"method_type": model.MethodTypeJWT,
							"source":      jwtSourceName,
						})))),

			expected: expectedWithLeaways(
				expectedWithUser(
					expectedWithBoundClaims(
						expectedWithTokenParams(model.AuthMethod{
							MethodType: model.MethodTypeJWT,
							Source:     jwtSourceName,
							Name:       model.MethodTypeJWT,
						})))),

			updateBody: map[string]interface{}{
				"method_type": model.MethodTypeJWT,
				"source":      jwtSourceName,

				"bound_audiences":   "new",
				"user_claim":        "new",
				"expiration_leeway": "6s",
				"token_period":      "10s",
			},

			updateExpected: func(m model.AuthMethod) model.AuthMethod {
				m.BoundAudiences = []string{"new"}
				m.UserClaim = "new"
				m.ExpirationLeeway = 6 * time.Second
				m.TokenPeriod = 10 * time.Second
				return m
			}(expectedWithLeaways(
				expectedWithUser(
					expectedWithBoundClaims(
						expectedWithTokenParams(model.AuthMethod{
							MethodType: model.MethodTypeJWT,
							Source:     jwtSourceName,
							Name:       model.MethodTypeJWT,
						}))))),
		},

		{
			methodName: model.MethodTypeMultipass,
			body: withVaultTokenParts(map[string]interface{}{
				"method_type": model.MethodTypeMultipass,
			}),

			expected: expectedWithTokenParams(model.AuthMethod{
				MethodType: model.MethodTypeMultipass,
				Name:       model.MethodTypeMultipass,
			}),

			updateBody: map[string]interface{}{
				"method_type":       model.MethodTypeMultipass,
				"token_period":      "10s",
				"expiration_leeway": "6s",
			},

			updateExpected: func(m model.AuthMethod) model.AuthMethod {
				m.TokenPeriod = 10 * time.Second
				m.ExpirationLeeway = 6 * time.Second

				return m
			}(expectedWithTokenParams(model.AuthMethod{
				MethodType: model.MethodTypeMultipass,
				Name:       model.MethodTypeMultipass,
			})),
		},

		{
			methodName: model.MethodTypeSAPassword,
			body: withVaultTokenParts(map[string]interface{}{
				"method_type": model.MethodTypeSAPassword,
			}),
			expected: expectedWithTokenParams(model.AuthMethod{
				MethodType: model.MethodTypeSAPassword,
				Name:       model.MethodTypeSAPassword,
			}),

			updateBody: map[string]interface{}{
				"method_type":  model.MethodTypeSAPassword,
				"token_period": "10s",
			},

			updateExpected: func(m model.AuthMethod) model.AuthMethod {
				m.TokenPeriod = 10 * time.Second

				return m
			}(expectedWithTokenParams(model.AuthMethod{
				MethodType: model.MethodTypeSAPassword,
				Name:       model.MethodTypeSAPassword,
			})),
		},

		{
			methodName: model.MethodTypeOIDC,
			body: withBoundClaims(
				withLeaways(
					withVaultTokenParts(
						withUserClaims(map[string]interface{}{
							"method_type":           model.MethodTypeOIDC,
							"source":                oidcSourceName,
							"oidc_scopes":           []string{"email", "profile"},
							"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
							"max_age":               "5s",
						})))),
			expected: expectedWithUser(
				expectedWithBoundClaims(
					expectedWithTokenParams(model.AuthMethod{
						MethodType:          model.MethodTypeOIDC,
						Source:              oidcSourceName,
						Name:                model.MethodTypeOIDC,
						AllowedRedirectURIs: []string{"https://example.com", "http://localhost:8250"},
						OIDCScopes:          []string{"email", "profile"},
						MaxAge:              5 * time.Second,
					}))),

			updateBody: map[string]interface{}{
				"method_type": model.MethodTypeOIDC,
				"source":      oidcSourceName,

				"bound_audiences":   "new",
				"user_claim":        "new",
				"expiration_leeway": "6s",
				"token_period":      "10s",

				"oidc_scopes":           []string{"email"},
				"allowed_redirect_uris": []string{"http://localhost:8250"},
				"max_age":               "3s",
			},

			updateExpected: func(m model.AuthMethod) model.AuthMethod {
				m.BoundAudiences = []string{"new"}
				m.UserClaim = "new"
				m.TokenPeriod = 10 * time.Second

				m.MaxAge = 3 * time.Second
				m.OIDCScopes = []string{"email"}
				m.AllowedRedirectURIs = []string{"http://localhost:8250"}

				return m
			}(expectedWithUser(
				expectedWithBoundClaims(
					expectedWithTokenParams(model.AuthMethod{
						MethodType: model.MethodTypeOIDC,
						Source:     oidcSourceName,
						Name:       model.MethodTypeOIDC,
					})))),
		},
	}

	for _, m := range methods {
		t.Run(fmt.Sprintf("create happy path for %s", m.methodName), func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      fmt.Sprintf("auth_method/%s", m.methodName),
				Storage:   storage,
				Data:      m.body,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}

			assertAuthMethod(t, b, m.methodName, m.expected)

			t.Run(fmt.Sprintf("update happy path for %s", m.methodName), func(t *testing.T) {
				req := &logical.Request{
					Operation: logical.UpdateOperation,
					Path:      fmt.Sprintf("auth_method/%s", m.methodName),
					Storage:   storage,
					Data:      m.updateBody,
				}

				resp, err := b.HandleRequest(context.Background(), req)
				if err != nil || (resp != nil && resp.IsError()) {
					t.Fatalf("err:%s resp:%#v\n", err, resp)
				}

				assertAuthMethod(t, b, m.methodName, m.updateExpected)
			})
		})
	}
}

func TestAuthMethod_IncorrectUpdate(t *testing.T) {
	b, storage := getBackend(t)

	jwtSourceName := "a"
	enableJwtBackend(t, b, storage)
	creteTestJWTBasedSource(t, b, storage, jwtSourceName)

	cases := []struct {
		title      string
		updateBody map[string]interface{}
		methodName string
	}{
		{
			title:      "does not change method type when update",
			methodName: "a",
			updateBody: map[string]interface{}{
				"method_type": model.MethodTypeOIDC,
				"source":      jwtSourceName,
			},
		},

		{
			title:      "does not change source when update",
			methodName: "a",
			updateBody: map[string]interface{}{
				"method_type": model.MethodTypeJWT,
				"source":      "b",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      fmt.Sprintf("auth_method/%s", c.methodName),
				Storage:   storage,
				Data: withBoundClaims(
					withLeaways(
						withVaultTokenParts(
							withUserClaims(map[string]interface{}{
								"method_type": model.MethodTypeJWT,
								"source":      jwtSourceName,
							})))),
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}

			req = &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      fmt.Sprintf("auth_method/%s", c.methodName),
				Storage:   storage,
				Data:      c.updateBody,
			}

			resp, err = b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatalf("err:%s or response is nil", err)
			}

			if resp == nil || !resp.IsError() {
				t.Fatal("expected error")
			}
		})
	}
}

func TestAuthMethod_Read(t *testing.T) {
	b, storage := getBackend(t)

	jwtSourceName := "a"
	creteTestJWTBasedSource(t, b, storage, jwtSourceName)

	body := withBoundClaims(
		withLeaways(
			withVaultTokenParts(
				withUserClaims(map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"source":      jwtSourceName,
				}))))

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "auth_method/test",
		Storage:   storage,
		Data:      body,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	cidrsObj, err := parseutil.ParseAddrs([]string{"127.0.0.1/8"})
	if err != nil {
		panic(err)
	}

	expected := map[string]interface{}{
		"name": "test",
		"method_type":       model.MethodTypeJWT,
		"bound_claims_type": "glob",
		"bound_claims": map[string]interface{}{
			"foo": []interface{}{"baz"},
		},
		"claim_mappings": map[string]string{
			"foo": "a",
		},
		"bound_subject":           "testsub",
		"bound_audiences":         []string{"vault"},
		"allowed_redirect_uris":   []string(nil),
		"oidc_scopes":             []string(nil),
		"user_claim":              "user",
		"groups_claim":            "groups",
		"token_policies":          []string{"good"},
		"token_period":            int64(10),
		"token_ttl":               int64(5),
		"token_num_uses":          5,
		"token_max_ttl":           int64(100),
		"expiration_leeway":       int64(5),
		"not_before_leeway":       int64(5),
		"clock_skew_leeway":       int64(5),
		"verbose_oidc_logging":    false,
		"token_type":              logical.TokenTypeDefault.String(),
		"token_no_default_policy": true,
		"token_explicit_max_ttl":  int64(100),
		"max_age":                 int64(0),
		"token_bound_cidrs":       cidrsObj,
	}

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "auth_method/test",
		Storage:   storage,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	if diff := deep.Equal(expected, resp.Data); diff != nil {
		t.Fatal(diff)
	}
}

func TestAuthMethod_Delete(t *testing.T) {
	b, storage := getBackend(t)

	jwtSourceName := "a"
	creteTestJWTBasedSource(t, b, storage, jwtSourceName)

	body := withBoundClaims(
		withLeaways(
			withVaultTokenParts(
				withUserClaims(map[string]interface{}{
					"method_type": model.MethodTypeJWT,
					"source":      jwtSourceName,
				}))))

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "auth_method/test",
		Storage:   storage,
		Data:      body,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	req = &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "auth_method/test",
		Storage:   storage,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	if resp != nil {
		t.Fatalf("Unexpected resp data: expected nil got %#v\n", resp.Data)
	}

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "auth_method/test",
		Storage:   storage,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	if resp != nil {
		t.Fatalf("Unexpected resp data: expected nil got %#v\n", resp.Data)
	}
}
