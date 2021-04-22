package jwtauth

import (
	"context"
	"encoding/json"
	"github.com/go-test/deep"
	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/vault/sdk/helper/tokenutil"
	"reflect"
	"strings"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
)

func getBackend(t *testing.T) (logical.Backend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	return b, config.StorageView
}

func TestAuthMethod_Create(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":     methodTypeJWT,
			"bound_subject":   "testsub",
			"bound_audiences": "vault",
			"user_claim":      "user",
			"groups_claim":    "groups",
			"bound_cidrs":     "127.0.0.1/8",
			"source":          sourceName,
		}

		expected := &authMethodConfig{
			TokenParams: tokenutil.TokenParams{
				TokenPolicies:   []string{},
				TokenBoundCIDRs: []*sockaddr.SockAddrMarshaler{},
			},
			MethodType:      methodTypeJWT,
			BoundSubject:    "testsub",
			BoundAudiences:  []string{"vault"},
			BoundClaimsType: "string",
			UserClaim:       "user",
			GroupsClaim:     "groups",
			Source:          sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/plugin-test",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%s resp:%#v\n", err, resp)
		}
		actual, err := b.(*flantIamAuthBackend).authMethod(context.Background(), NewPrefixStorage("method/", storage), "plugin-test")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Unexpected authMethod data: expected %#v\n got %#v\n", expected, actual)
		}
	})

	t.Run("oidc need oids source jwt need jwt source", func(t *testing.T) {
		b, storage := getBackend(t)

		methods := map[string]func(*testing.T, logical.Backend, logical.Storage, string){
			methodTypeOIDC: func(t *testing.T, b logical.Backend, s logical.Storage, n string) {
				creteTestJWTBasedSource(t, b, s, n)
			},

			methodTypeJWT: func(t *testing.T, b logical.Backend, s logical.Storage, n string) {
				creteTestOIDCBasedSource(t, b, s, n)
			},
		}

		for methodType, sourceCreator := range methods {
			sourceName := "a"
			sourceCreator(t, b, storage, sourceName)

			data := map[string]interface{}{
				"method_type":     methodType,
				"bound_subject":   "testsub",
				"bound_audiences": "vault",
				"user_claim":      "user",
				"groups_claim":    "groups",
				"bound_cidrs":     "127.0.0.1/8",
				"source":          sourceName,
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/plugin-test",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatalf("err:%v\n", err)
			}

			if resp == nil || !resp.IsError() || !strings.Contains(resp.Error().Error(), "incorrect source") {
				t.Fatalf("need incorect source error")
			}
		}

	})

	t.Run("need source name for jwt and oidc", func(t *testing.T) {
		b, storage := getBackend(t)

		for _, methodType := range []string{methodTypeOIDC, methodTypeJWT} {
			data := map[string]interface{}{
				"method_type":     methodType,
				"bound_subject":   "testsub",
				"bound_audiences": "vault",
				"user_claim":      "user",
				"groups_claim":    "groups",
				"bound_cidrs":     "127.0.0.1/8",
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/plugin-test-source",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || resp == nil {
				t.Fatalf("err:%s or response is nil", err)
			}

			if resp.Error().Error() != "missing source" {
				t.Fatalf("must return need source name error")
			}
		}

	})

	t.Run("check source is exists for jwt and oidc", func(t *testing.T) {
		b, storage := getBackend(t)

		for _, methodType := range []string{methodTypeOIDC, methodTypeJWT} {
			data := map[string]interface{}{
				"method_type":     methodType,
				"bound_subject":   "testsub",
				"bound_audiences": "vault",
				"user_claim":      "user",
				"groups_claim":    "groups",
				"bound_cidrs":     "127.0.0.1/8",
				"source":          "no",
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/plugin-test-source-no",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || resp == nil {
				t.Fatalf("err:%s or response is nil", err)
			}

			if resp.Error().Error() != "'no': auth source not found" {
				t.Fatalf("must return no auth source error")
			}
		}

	})

	t.Run("no user claim", func(t *testing.T) {
		b, storage := getBackend(t)
		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"policies":    "test",
			"method_type": methodTypeJWT,
			"source":      sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test2",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if resp.Error().Error() != "a user claim must be defined on the authMethodConfig" {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

	t.Run("no binding", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type": methodTypeJWT,
			"user_claim":  "user",
			"policies":    "test",
			"source":      sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test3",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if !strings.HasPrefix(resp.Error().Error(), "must have at least one bound constraint") {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

	t.Run("has bound subject", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":   methodTypeJWT,
			"user_claim":    "user",
			"policies":      "test",
			"bound_subject": "testsub",
			"source":        sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test4",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && resp.IsError() {
			t.Fatalf("did not expect error")
		}
	})

	t.Run("has audience", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":     methodTypeJWT,
			"user_claim":      "user",
			"policies":        "test",
			"bound_audiences": "vault",
			"source":          sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test5",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && resp.IsError() {
			t.Fatalf("did not expect error")
		}
	})

	t.Run("has cidr", func(t *testing.T) {
		b, storage := getBackend(t)

		for _, methodType := range []string{methodTypeOwn, methodTypeSAPassword} {
			data := map[string]interface{}{
				"method_type":       methodType,
				"user_claim":        "user",
				"policies":          "test",
				"token_bound_cidrs": "127.0.0.1/8",
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/test6",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if resp != nil && resp.IsError() {
				t.Fatalf("did not expect error")
			}
		}
	})

	t.Run("has bound claims", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type": methodTypeJWT,
			"user_claim":  "user",
			"policies":    "test",
			"bound_claims": map[string]interface{}{
				"foo": 10,
				"bar": "baz",
			},
			"source": sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test7",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && resp.IsError() {
			t.Fatalf("did not expect error")
		}
	})

	t.Run("has expiration, not before custom leeways for own type auth", func(t *testing.T) {
		b, storage := getBackend(t)

		for _, methodType := range []string{methodTypeOwn, methodTypeSAPassword} {
			data := map[string]interface{}{
				"method_type":       methodType,
				"user_claim":        "user",
				"policies":          "test",
				"expiration_leeway": "5s",
				"not_before_leeway": "5s",
				"clock_skew_leeway": "5s",
				"bound_claims": map[string]interface{}{
					"foo": 10,
					"bar": "baz",
				},
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/test8",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if resp != nil && resp.IsError() {
				t.Fatalf("did not expect error:%s", resp.Error().Error())
			}

			actual, err := b.(*flantIamAuthBackend).authMethod(context.Background(), NewPrefixStorage("method/", storage), "test8")
			if err != nil {
				t.Fatal(err)
			}

			expectedDuration := "5s"
			if actual.ExpirationLeeway.String() != expectedDuration {
				t.Fatalf("expiration_leeway - expected: %s, got: %s", expectedDuration, actual.ExpirationLeeway)
			}

			if actual.NotBeforeLeeway.String() != expectedDuration {
				t.Fatalf("not_before_leeway - expected: %s, got: %s", expectedDuration, actual.NotBeforeLeeway)
			}

			if actual.ClockSkewLeeway.String() != expectedDuration {
				t.Fatalf("clock_skew_leeway - expected: %s, got: %s", expectedDuration, actual.ClockSkewLeeway)
			}
		}

	})

	t.Run("storing zero leeways for jwt and oidc and sa", func(t *testing.T) {
		methods := map[string]func(*testing.T, logical.Backend, logical.Storage, string){
			methodTypeOIDC: func(t *testing.T, b logical.Backend, s logical.Storage, n string) {
				creteTestOIDCBasedSource(t, b, s, n)
			},
			methodTypeJWT: func(t *testing.T, b logical.Backend, s logical.Storage, n string) {
				creteTestJWTBasedSource(t, b, s, n)
			},
			methodTypeSAPassword: func(*testing.T, logical.Backend, logical.Storage, string) {},
		}

		for methodType, sourceCreator := range methods {
			b, storage := getBackend(t)

			sourceName := "a"
			sourceCreator(t, b, storage, sourceName)

			data := map[string]interface{}{
				"method_type": methodType,
				"user_claim":  "user",
				"policies":    "test",
				"bound_claims": map[string]interface{}{
					"foo": 10,
					"bar": "baz",
				},

				"expiration_leeway":     "5s",
				"not_before_leeway":     "5s",
				"clock_skew_leeway":     "5s",
				"allowed_redirect_uris": []string{"https://example.com"},
				"source":                sourceName,
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/test9",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if resp != nil && resp.IsError() {
				t.Fatalf("did not expect error:%s", resp.Error().Error())
			}

			actual, err := b.(*flantIamAuthBackend).authMethod(context.Background(), NewPrefixStorage("method/", storage), "test9")
			if err != nil {
				t.Fatal(err)
			}

			if actual.ClockSkewLeeway.Seconds() != 0 {
				t.Fatalf("clock_skew_leeway - expected: 0, got: %v", actual.ClockSkewLeeway.Seconds())
			}
			if actual.ExpirationLeeway.Seconds() != 0 {
				t.Fatalf("expiration_leeway - expected: 0, got: %v", actual.ExpirationLeeway.Seconds())
			}
			if actual.NotBeforeLeeway.Seconds() != 0 {
				t.Fatalf("not_before_leeway - expected: 0, got: %v", actual.NotBeforeLeeway.Seconds())
			}
		}

	})

	t.Run("storing negative leeways", func(t *testing.T) {
		b, storage := getBackend(t)

		for _, methodType := range []string{methodTypeOwn, methodTypeSAPassword} {
			data := map[string]interface{}{
				"method_type":       methodType,
				"user_claim":        "user",
				"policies":          "test",
				"clock_skew_leeway": "-1",
				"expiration_leeway": "-1",
				"not_before_leeway": "-1",
				"bound_claims": map[string]interface{}{
					"foo": 10,
					"bar": "baz",
				},
			}

			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/test9",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if resp != nil && resp.IsError() {
				t.Fatalf("did not expect error:%s", resp.Error().Error())
			}

			actual, err := b.(*flantIamAuthBackend).authMethod(context.Background(), NewPrefixStorage("method/", storage), "test9")
			if err != nil {
				t.Fatal(err)
			}

			if actual.ClockSkewLeeway.Seconds() != -1 {
				t.Fatalf("clock_skew_leeway - expected: -1, got: %v", actual.ClockSkewLeeway.Seconds())
			}
			if actual.ExpirationLeeway.Seconds() != -1 {
				t.Fatalf("expiration_leeway - expected: -1, got: %v", actual.ExpirationLeeway.Seconds())
			}
			if actual.NotBeforeLeeway.Seconds() != -1 {
				t.Fatalf("not_before_leeway - expected: -1, got: %v", actual.NotBeforeLeeway.Seconds())
			}
		}
	})

	t.Run("storing an invalid bound_claim_type", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":       methodTypeJWT,
			"user_claim":        "user",
			"policies":          "test",
			"bound_claims_type": "invalid",
			"bound_claims": map[string]interface{}{
				"foo": 10,
				"bar": "baz",
			},
			"source": sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test10",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if resp.Error().Error() != "invalid 'bound_claims_type': invalid" {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

	t.Run("with invalid glob in claim", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":       methodTypeJWT,
			"user_claim":        "user",
			"policies":          "test",
			"bound_claims_type": "glob",
			"bound_claims": map[string]interface{}{
				"bar": "baz",
				"foo": 25,
			},

			"source": sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test11",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if resp.Error().Error() != "claim is not a string or list: 25" {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

	t.Run("authMethod with invalid glob in claim array", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "a"
		creteTestJWTBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":       methodTypeJWT,
			"user_claim":        "user",
			"policies":          "test",
			"clock_skew_leeway": "-1",
			"expiration_leeway": "-1",
			"not_before_leeway": "-1",
			"bound_claims_type": "glob",
			"bound_claims": map[string]interface{}{
				"foo": []interface{}{"baz", 10},
			},

			"source": sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test12",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if resp.Error().Error() != "claim is not a string: 10" {
			t.Fatalf("unexpected err: %v", resp)
		}
	})
}

func TestAuthMethod_OIDCCreate(t *testing.T) {
	t.Run("create oidc", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "b"
		creteTestOIDCBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"bound_audiences": "vault",
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
			"source":       sourceName,
		}

		expected := &authMethodConfig{
			TokenParams: tokenutil.TokenParams{
				TokenPolicies:   []string{},
				TokenBoundCIDRs: []*sockaddr.SockAddrMarshaler{},
			},
			MethodType:      "oidc",
			BoundAudiences:  []string{"vault"},
			BoundClaimsType: "string",
			BoundClaims: map[string]interface{}{
				"foo": json.Number("10"),
				"bar": "baz",
			},
			AllowedRedirectURIs: []string{"https://example.com", "http://localhost:8250"},
			ClaimMappings: map[string]string{
				"foo": "a",
				"bar": "b",
			},
			OIDCScopes:  []string{"email", "profile"},
			UserClaim:   "user",
			GroupsClaim: "groups",
			Source:      sourceName,
		}

		for _, methodType := range []string{methodTypeOIDC} {
			data["method_type"] = methodType
			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "auth_method/plugin-test",
				Storage:   storage,
				Data:      data,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}
			actual, err := b.(*flantIamAuthBackend).authMethod(context.Background(), NewPrefixStorage("method/", storage), "plugin-test")
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(expected, actual); diff != nil {
				t.Fatal(diff)
			}
		}
	})

	t.Run("invalid reserved metadata key authMethod", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "b"
		creteTestOIDCBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":     methodTypeOIDC,
			"bound_audiences": "vault",
			"bound_claims": map[string]interface{}{
				"foo": 10,
				"bar": "baz",
			},
			"oidc_scopes":           []string{"email", "profile"},
			"allowed_redirect_uris": []string{"https://example.com", "http://localhost:8250"},
			"claim_mappings": map[string]string{
				"foo":        "a",
				"some_claim": "authMethodConfig",
			},
			"user_claim":   "user",
			"groups_claim": "groups",
			"source":       sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test2",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if !strings.Contains(resp.Error().Error(), `metadata key "authMethodConfig" is reserved`) {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

	t.Run("invalid duplicate metadata destination", func(t *testing.T) {
		b, storage := getBackend(t)

		sourceName := "b"
		creteTestOIDCBasedSource(t, b, storage, sourceName)

		data := map[string]interface{}{
			"method_type":     methodTypeOIDC,
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
			"user_claim":        "user",
			"groups_claim":      "groups",
			"policies":          "test",
			"period":            "3s",
			"ttl":               "1s",
			"num_uses":          12,
			"max_ttl":           "5s",
			"expiration_leeway": "300s",
			"not_before_leeway": "300s",
			"clock_skew_leeway": "1s",
			"source":            sourceName,
		}

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "auth_method/test2",
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && !resp.IsError() {
			t.Fatalf("expected error")
		}
		if !strings.Contains(resp.Error().Error(), `multiple keys are mapped to metadata key "a"`) {
			t.Fatalf("unexpected err: %v", resp)
		}
	})

}

//func TestAuthMethod_Read(t *testing.T) {
//	b, storage := getBackend(t)
//
//	data := map[string]interface{}{
//		"method_type":             "jwt",
//		"bound_subject":         "testsub",
//		"bound_audiences":       "vault",
//		"allowed_redirect_uris": []string{"http://127.0.0.1"},
//		"oidc_scopes":           []string{"email", "profile"},
//		"user_claim":            "user",
//		"groups_claim":          "groups",
//		"bound_cidrs":           "127.0.0.1/8",
//		"policies":              "test",
//		"period":                "3s",
//		"ttl":                   "1s",
//		"num_uses":              12,
//		"max_ttl":               "5s",
//		"expiration_leeway":     "500s",
//		"not_before_leeway":     "500s",
//		"clock_skew_leeway":     "100s",
//	}
//
//	expected := map[string]interface{}{
//		"method_type":               "jwt",
//		"bound_claims_type":       "string",
//		"bound_claims":            map[string]interface{}(nil),
//		"claim_mappings":          map[string]string(nil),
//		"bound_subject":           "testsub",
//		"bound_audiences":         []string{"vault"},
//		"allowed_redirect_uris":   []string{"http://127.0.0.1"},
//		"oidc_scopes":             []string{"email", "profile"},
//		"user_claim":              "user",
//		"groups_claim":            "groups",
//		"token_policies":          []string{"test"},
//		"policies":                []string{"test"},
//		"token_period":            int64(3),
//		"period":                  int64(3),
//		"token_ttl":               int64(1),
//		"ttl":                     int64(1),
//		"token_num_uses":          12,
//		"num_uses":                12,
//		"token_max_ttl":           int64(5),
//		"max_ttl":                 int64(5),
//		"expiration_leeway":       int64(500),
//		"not_before_leeway":       int64(500),
//		"clock_skew_leeway":       int64(100),
//		"verbose_oidc_logging":    false,
//		"token_type":              logical.TokenTypeDefault.String(),
//		"token_no_default_policy": false,
//		"token_explicit_max_ttl":  int64(0),
//		"max_age":                 int64(0),
//	}
//
//	req := &logical.Request{
//		Operation: logical.CreateOperation,
//		Path:      "auth_method/plugin-test",
//		Storage:   storage,
//		Data:      data,
//	}
//
//	resp, err := b.HandleRequest(context.Background(), req)
//	if err != nil || (resp != nil && resp.IsError()) {
//		t.Fatalf("err:%s resp:%#v\n", err, resp)
//	}
//
//	readTest := func() {
//		req = &logical.Request{
//			Operation: logical.ReadOperation,
//			Path:      "auth_method/plugin-test",
//			Storage:   storage,
//		}
//
//		resp, err = b.HandleRequest(context.Background(), req)
//		if err != nil || (resp != nil && resp.IsError()) {
//			t.Fatalf("err:%s resp:%#v\n", err, resp)
//		}
//
//		if resp.Data["bound_cidrs"].([]*sockaddr.SockAddrMarshaler)[0].String() != "127.0.0.1/8" {
//			t.Fatal("unexpected bound cidrs")
//		}
//		delete(resp.Data, "bound_cidrs")
//		if resp.Data["token_bound_cidrs"].([]*sockaddr.SockAddrMarshaler)[0].String() != "127.0.0.1/8" {
//			t.Fatal("unexpected token bound cidrs")
//		}
//		delete(resp.Data, "token_bound_cidrs")
//		if diff := deep.Equal(expected, resp.Data); diff != nil {
//			t.Fatal(diff)
//		}
//	}
//
//	// Run read test for normal case
//	readTest()
//
//	// Remove the 'method_type' parameter in stored authMethodConfig to simulate a legacy authMethodConfig
//	rolePath := rolePrefix + "plugin-test"
//	raw, err := storage.Get(context.Background(), rolePath)
//
//	var role map[string]interface{}
//	if err := raw.DecodeJSON(&role); err != nil {
//		t.Fatal(err)
//	}
//	delete(role, "method_type")
//	entry, err := logical.StorageEntryJSON(rolePath, role)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if err = req.Storage.Put(context.Background(), entry); err != nil {
//		t.Fatal(err)
//	}
//
//	// Run read test for "upgrade" case. The legacy authMethodConfig is not changed in storage, but
//	// reads will populate the `method_type` with "jwt".
//	readTest()
//
//	// Remove the 'bound_claims_type' parameter in stored authMethodConfig to simulate a legacy authMethodConfig
//	raw, err = storage.Get(context.Background(), rolePath)
//
//	if err := raw.DecodeJSON(&role); err != nil {
//		t.Fatal(err)
//	}
//	delete(role, "bound_claims_type")
//	entry, err = logical.StorageEntryJSON(rolePath, role)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if err = req.Storage.Put(context.Background(), entry); err != nil {
//		t.Fatal(err)
//	}
//
//	// Run read test for "upgrade" case. The legacy authMethodConfig is not changed in storage, but
//	// reads will populate the `bound_claims_type` with "string".
//	readTest()
//}

func TestAuthMethod_Delete(t *testing.T) {
	b, storage := getBackend(t)

	sourceName := "a"
	creteTestJWTBasedSource(t, b, storage, sourceName)

	data := map[string]interface{}{
		"method_type":       methodTypeJWT,
		"bound_subject":     "testsub",
		"bound_audiences":   "vault",
		"user_claim":        "user",
		"groups_claim":      "groups",
		"bound_cidrs":       "127.0.0.1/8",
		"policies":          "test",
		"period":            "3s",
		"ttl":               "1s",
		"num_uses":          12,
		"max_ttl":           "5s",
		"expiration_leeway": "300s",
		"not_before_leeway": "300s",
		"source":            sourceName,
	}

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "auth_method/plugin-test",
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	req = &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "auth_method/plugin-test",
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
		Path:      "auth_method/plugin-test",
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
