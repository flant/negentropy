package jwtauth

import (
	"context"
	"crypto"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/vault/sdk/helper/certutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
)

const authSourceTestName = "a"

func getAuthSourcePath(name string) string {
	return fmt.Sprintf("auth_source/%s", name)
}

var authSourceTestPath = getAuthSourcePath(authSourceTestName)

func creteTestJWTBasedSource(t *testing.T, b logical.Backend, storage logical.Storage, name string) (*logical.Response, map[string]interface{}) {
	// Create a config with too many token verification schemes
	data := map[string]interface{}{
		"jwt_validation_pubkeys": []string{testJWTPubKey},
		"bound_issuer":           "http://vault.example.com/",
		"entity_alias_name":      model.EntityAliasNameEmail,
		"allow_service_accounts": false,
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      getAuthSourcePath(name),
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("not create source err:%v resp:%v", err, resp.Error())
	}

	return resp, data
}

func creteTestOIDCBasedSource(t *testing.T, b logical.Backend, storage logical.Storage, name string) (*logical.Response, map[string]interface{}) {
	// First we provide an invalid CA cert to verify that it is in fact paying
	// attention to the value we specify
	data := map[string]interface{}{
		"oidc_discovery_url":     "https://team-vault.auth0.com/",
		"oidc_client_id":         "abc",
		"oidc_client_secret":     "def",
		"entity_alias_name":      model.EntityAliasNameEmail,
		"allow_service_accounts": false,
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      getAuthSourcePath(name),
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || resp.IsError() {
		t.Fatal("not create source")
	}

	return resp, data
}

func assertAuthSourceErrorCases(t *testing.T, errorCases []struct {
	title     string
	body      map[string]interface{}
	errPrefix string
}) {
	b, storage := getBackend(t)
	for _, c := range errorCases {
		t.Run(fmt.Sprintf("does not update or create because %v", c.title), func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      authSourceTestPath,
				Storage:   storage,
				Data:      c.body,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil {
				t.Fatal(err)
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

func assertAuthSource(t *testing.T, b *flantIamAuthBackend, name string, expected *model.AuthSource) {
	conf, err := repo.NewAuthSourceRepo(b.storage.Txn(false)).Get(name)
	if err != nil {
		t.Fatal(err)
	}

	if conf.UUID == "" {
		t.Fatal("uuid must be not empty")
	}

	conf.UUID = ""
	if diff := deep.Equal(expected, conf); diff != nil {
		t.Fatal(diff)
	}
}

func TestAuthSource_Read(t *testing.T) {
	b, storage := getBackend(t)

	data := map[string]interface{}{
		"oidc_discovery_url":     "",
		"oidc_discovery_ca_pem":  "",
		"oidc_client_id":         "",
		"oidc_response_mode":     "",
		"oidc_response_types":    []string{},
		"default_role":           "",
		"jwt_validation_pubkeys": []string{testJWTPubKey},
		"jwt_supported_algs":     []string{},
		"jwks_url":               "",
		"jwks_ca_pem":            "",
		"bound_issuer":           "http://vault.example.com/",
		"namespace_in_state":     false,
		"entity_alias_name":      model.EntityAliasNameEmail,
		"allow_service_accounts": false,
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      nil,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	if diff := deep.Equal(resp.Data, data); diff != nil {
		t.Fatalf("Expected did not equal actual: %v", diff)
	}
}

func TestAuthSource_LogicalErrorUpdate(t *testing.T) {
	errorCases := []struct {
		title     string
		body      map[string]interface{}
		errPrefix string
	}{
		{
			title: "incorrect entity alias name",
			body: map[string]interface{}{
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      "Incorrect",
				"allow_service_accounts": false,
			},

			errPrefix: "incorrect entity_alias_name",
		},

		{
			title: "entity alias name with allow service accounts",
			body: map[string]interface{}{
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": true,
			},

			errPrefix: "conflict values for entity_alias_name and allow_service_accounts",
		},

		{
			title: "oidc and jwks url and jwks ain one time",
			body: map[string]interface{}{
				"oidc_discovery_url":     "http://fake.example.com",
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"jwks_url":               "http://fake.anotherexample.com",
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			errPrefix: "exactly one of",
		},

		{
			title: "oidc and jwks ain one time",
			body: map[string]interface{}{
				"oidc_discovery_url":     "http://fake.example.com",
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			errPrefix: "exactly one of",
		},

		{
			title: "jwks url and jwks ain one time",
			body: map[string]interface{}{
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"jwks_url":               "http://fake.anotherexample.com",
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			errPrefix: "exactly one of",
		},
	}

	assertAuthSourceErrorCases(t, errorCases)
}

func TestAuthSource_JWTUpdate(t *testing.T) {
	b, storage := getBackend(t)

	errorCases := []struct {
		title     string
		body      map[string]interface{}
		errPrefix string
	}{
		{
			title: "invalid pub keys",
			body: map[string]interface{}{
				"jwt_validation_pubkeys": []string{"invalid"},
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			errPrefix: "error parsing public key",
		},

		{
			title: "invalid jwt supported algs",
			body: map[string]interface{}{
				"jwt_validation_pubkeys": []string{testJWTPubKey},
				"bound_issuer":           "http://vault.example.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
				"jwt_supported_algs":     []string{"invalid"},
			},
			errPrefix: "unsupported signing algorithm",
		},
	}

	assertAuthSourceErrorCases(t, errorCases)

	t.Run("creates auth source with jwt pub keys", func(t *testing.T) {
		data := map[string]interface{}{
			"jwt_validation_pubkeys": []string{testJWTPubKey},
			"bound_issuer":           "http://vault.example.com/",
			"entity_alias_name":      model.EntityAliasNameEmail,
			"allow_service_accounts": false,
		}

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%s resp:%#v\n", err, resp)
		}

		pubkey, err := certutil.ParsePublicKeyPEM([]byte(testJWTPubKey))
		if err != nil {
			t.Fatal(err)
		}

		expected := &model.AuthSource{
			Name: authSourceTestName,

			ParsedJWTPubKeys:     []crypto.PublicKey{pubkey},
			JWTValidationPubKeys: []string{testJWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			EntityAliasName:      model.EntityAliasNameEmail,
		}

		assertAuthSource(t, b, authSourceTestName, expected)
	})
}

func TestAuthSource_JWKS_Update(t *testing.T) {
	b, storage := getBackend(t)

	s := newOIDCProvider(t)
	defer s.server.Close()

	cert, err := s.getTLSCert()
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{
		"jwks_url":               s.server.URL + "/certs",
		"jwks_ca_pem":            cert,
		"entity_alias_name":      model.EntityAliasNameEmail,
		"allow_service_accounts": false,
		"namespace_in_state":     false,
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      data,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	expected := &model.AuthSource{
		Name: authSourceTestName,

		JWKSURL:              s.server.URL + "/certs",
		JWKSCAPEM:            cert,
		EntityAliasName:      model.EntityAliasNameEmail,
		AllowServiceAccounts: false,
		NamespaceInState:     false,

		OIDCResponseTypes:    []string{},
		JWTSupportedAlgs:     []string{},
		JWTValidationPubKeys: []string{},
	}

	assertAuthSource(t, b, authSourceTestName, expected)
}

func TestAuthSource_JWKS_Update_Invalid(t *testing.T) {
	s := newOIDCProvider(t)
	defer s.server.Close()

	cert, err := s.getTLSCert()
	if err != nil {
		t.Fatal(err)
	}

	errorCases := []struct {
		title     string
		body      map[string]interface{}
		errPrefix string
	}{
		{
			title: "certs missing",
			body: map[string]interface{}{
				"jwks_url":               s.server.URL + "/certs_missing",
				"jwks_ca_pem":            cert,
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},

			errPrefix: "fetching keys oidc: get keys failed",
		},

		{
			title: "certs invalid",
			body: map[string]interface{}{
				"jwks_url":               s.server.URL + "/certs_invalid",
				"jwks_ca_pem":            cert,
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},

			errPrefix: "failed to decode keys",
		},
	}

	assertAuthSourceErrorCases(t, errorCases)
}

func TestAuthSource_ResponseMode(t *testing.T) {
	b, storage := getBackend(t)

	tests := []struct {
		mode        string
		errExpected bool
	}{
		{"", false},
		{"form_post", false},
		{"query", false},
		{"QUERY", true},
		{"abc", true},
	}

	for _, test := range tests {
		data := map[string]interface{}{
			"oidc_response_mode":     test.mode,
			"jwt_validation_pubkeys": []string{testJWTPubKey},
			"entity_alias_name":      model.EntityAliasNameEmail,
			"allow_service_accounts": false,
		}

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
			Data:      data,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if test.errExpected {
			if err == nil && (resp == nil || !resp.IsError()) {
				t.Fatalf("expected error, got none for %q", test.mode)
			}
		} else {
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}
		}
	}
}

func TestAuthSource_OIDC_Write(t *testing.T) {
	b, storage := getBackend(t)

	// First we provide an invalid CA cert to verify that it is in fact paying
	// attention to the value we specify
	data := map[string]interface{}{
		"oidc_discovery_url":     "https://team-vault.auth0.com/",
		"oidc_discovery_ca_pem":  oidcBadCACerts,
		"oidc_client_id":         "abc",
		"oidc_client_secret":     "def",
		"entity_alias_name":      model.EntityAliasNameEmail,
		"allow_service_accounts": false,
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      data,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsError() {
		t.Fatal("expected error")
	}

	delete(data, "oidc_discovery_ca_pem")

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	expected := &model.AuthSource{
		Name: authSourceTestName,

		JWTValidationPubKeys: []string{},
		JWTSupportedAlgs:     []string{},
		OIDCResponseTypes:    []string{},
		OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
		OIDCClientID:         "abc",
		OIDCClientSecret:     "def",
		NamespaceInState:     true,
		EntityAliasName:      model.EntityAliasNameEmail,
	}

	assertAuthSource(t, b, authSourceTestName, expected)

	// Verify OIDC config sanity:
	//   - if providing client id/secret, discovery URL needs to be set
	//   - both oidc client and secret should be provided if either one is
	tests := []struct {
		id   string
		data map[string]interface{}
	}{
		{
			"missing discovery URL",
			map[string]interface{}{
				"jwt_validation_pubkeys": []string{"a"},
				"oidc_client_id":         "abc",
				"oidc_client_secret":     "def",
			},
		},
		{
			"missing secret",
			map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"oidc_client_id":     "abc",
			},
		},
		{
			"missing ID",
			map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"oidc_client_secret": "abc",
			},
		},
	}

	for _, test := range tests {
		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
			Data:      test.data,
		}
		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("test '%s', %v", test.id, err)
		}
		if !resp.IsError() {
			t.Fatalf("test '%s', expected error", test.id)
		}
	}
}

func TestAuthSource_OIDC_Create_Namespace(t *testing.T) {
	type testCase struct {
		create   map[string]interface{}
		expected model.AuthSource
	}
	tests := map[string]testCase{
		"namespace_in_state not specified": {
			create: map[string]interface{}{
				"oidc_discovery_url":     "https://team-vault.auth0.com/",
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
		"namespace_in_state true": {
			create: map[string]interface{}{
				"oidc_discovery_url":     "https://team-vault.auth0.com/",
				"namespace_in_state":     true,
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
		"namespace_in_state false": {
			create: map[string]interface{}{
				"oidc_discovery_url":     "https://team-vault.auth0.com/",
				"namespace_in_state":     false,
				"entity_alias_name":      model.EntityAliasNameEmail,
				"allow_service_accounts": false,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b, storage := getBackend(t)

			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      authSourceTestPath,
				Storage:   storage,
				Data:      test.create,
			}
			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}

			assertAuthSource(t, b, authSourceTestName, &test.expected)
		})
	}
}

func TestAuthSource_OIDC_Update_Namespace(t *testing.T) {
	type testCase struct {
		existing map[string]interface{}
		update   map[string]interface{}
		expected model.AuthSource
	}
	tests := map[string]testCase{
		"existing false, update to true": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
		"existing false, update something else": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"default_role":       "ui",
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				DefaultRole:          "ui",
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
		"existing true, update to false": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
		"existing true, update something else": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"default_role":       "ui",
				"entity_alias_name":  model.EntityAliasNameEmail,
			},
			expected: model.AuthSource{
				Name: authSourceTestName,

				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				DefaultRole:          "ui",
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
				EntityAliasName:      model.EntityAliasNameEmail,
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b, storage := getBackend(t)

			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      authSourceTestPath,
				Storage:   storage,
				Data:      test.existing,
			}
			resp, err := b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}

			req.Data = test.update
			resp, err = b.HandleRequest(context.Background(), req)
			if err != nil || (resp != nil && resp.IsError()) {
				t.Fatalf("err:%s resp:%#v\n", err, resp)
			}

			assertAuthSource(t, b, authSourceTestName, &test.expected)
		})
	}
}

func TestAuthSource_Delete(t *testing.T) {
	t.Run("successful remove source", func(t *testing.T) {
		b, storage := getBackend(t)
		creteTestJWTBasedSource(t, b, storage, authSourceTestName)

		req := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%s resp:%#v\n", err, resp)
		}

		conf, err := repo.NewAuthSourceRepo(b.storage.Txn(false)).Get(authSourceTestName)
		if err != nil {
			t.Fatal(err)
		}
		if conf != nil {
			t.Fatal("source must be deleted")
		}
	})

	t.Run("delete not exists source must return error", func(t *testing.T) {
		b, storage := getBackend(t)

		req := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("err:%s resp:%#v\n", err, resp)
		}

		if !resp.IsError() {
			t.Fatal("must return error")
		}
	})

	t.Run("does not delete source if it uses in one more method", func(t *testing.T) {
		b, storage := getBackend(t)

		creteTestJWTBasedSource(t, b, storage, authSourceTestName)
		createJwtAuthMethod(t, b, storage, "jwt_method", authSourceTestName)

		req := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      authSourceTestPath,
			Storage:   storage,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("err:%s resp:%#v\n", err, resp)
		}

		if !resp.IsError() {
			t.Fatal("must return error")
		}
	})
}

const (
	testJWTPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEEVs/o5+uQbTjL3chynL4wXgUg2R9
q9UU8I5mEovUf86QZ7kOBIjJwqnzD1omageEHWwHdBO6B+dFabmdT9POxg==
-----END PUBLIC KEY-----`

	oidcBadCACerts = `-----BEGIN CERTIFICATE-----
MIIDYDCCAkigAwIBAgIJAK8uAVsPxWKGMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTgwNzA5MTgwODI5WhcNMjgwNzA2MTgwODI5WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA1eaEmIHKQqDlSadCtg6YY332qIMoeSb2iZTRhBRYBXRhMIKF3HoLXlI8
/3veheMnBQM7zxIeLwtJ4VuZVZcpJlqHdsXQVj6A8+8MlAzNh3+Xnv0tjZ83QLwZ
D6FWvMEzihxATD9uTCu2qRgeKnMYQFq4EG72AGb5094zfsXTAiwCfiRPVumiNbs4
Mr75vf+2DEhqZuyP7GR2n3BKzrWo62yAmgLQQ07zfd1u1buv8R72HCYXYpFul5qx
slZHU3yR+tLiBKOYB+C/VuB7hJZfVx25InIL1HTpIwWvmdk3QzpSpAGIAxWMXSzS
oRmBYGnsgR6WTymfXuokD4ZhHOpFZQIDAQABo1MwUTAdBgNVHQ4EFgQURh/QFJBn
hMXcgB1bWbGiU9B2VBQwHwYDVR0jBBgwFoAURh/QFJBnhMXcgB1bWbGiU9B2VBQw
DwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAr8CZLA3MQjMDWweS
ax9S1fRb8ifxZ4RqDcLj3dw5KZqnjEo8ggczR66T7vVXet/2TFBKYJAM0np26Z4A
WjZfrDT7/bHXseWQAUhw/k2d39o+Um4aXkGpg1Paky9D+ddMdbx1hFkYxDq6kYGd
PlBYSEiYQvVxDx7s7H0Yj9FWKO8WIO6BRUEvLlG7k/Xpp1OI6dV3nqwJ9CbcbqKt
ff4hAtoAmN0/x6yFclFFWX8s7bRGqmnoj39/r98kzeGFb/lPKgQjSVcBJuE7UO4k
8HP6vsnr/ruSlzUMv6XvHtT68kGC1qO3MfqiPhdSa4nxf9g/1xyBmAw/Uf90BJrm
sj9DpQ==
-----END CERTIFICATE-----`
)
