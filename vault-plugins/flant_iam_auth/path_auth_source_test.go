package jwtauth

import (
	"context"
	"crypto"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/vault/sdk/helper/certutil"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
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
		"oidc_discovery_url": "https://team-vault.auth0.com/",
		"oidc_client_id":     "abc",
		"oidc_client_secret": "def",
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

func TestAuthSource_WriteInStorage(t *testing.T) {
	b, storage := getBackend(t)

	_, data := creteTestJWTBasedSource(t, b, storage, authSourceTestName)

	dataFromStore, err := storage.Get(context.TODO(), fmt.Sprintf("%s/%s", "source", authSourceTestName))
	if err != nil {
		t.Fatalf("storage returns error %v", err)
	}
	if dataFromStore == nil {
		t.Fatal("storage returns nil data")
	} else {
		out := &authSource{}
		err = dataFromStore.DecodeJSON(out)
		if err != nil {
			t.Fatal("does not decode entry")
		}
		deepEq := reflect.DeepEqual(out.JWTValidationPubKeys, data["jwt_validation_pubkeys"])
		if !deepEq || out.BoundIssuer != data["bound_issuer"] {
			t.Fatal("does not decode entry")
		}
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

func TestAuthSource_JWTUpdate(t *testing.T) {
	b, storage := getBackend(t)

	// Create a config with too many token verification schemes
	data := map[string]interface{}{
		"oidc_discovery_url":     "http://fake.example.com",
		"jwt_validation_pubkeys": []string{testJWTPubKey},
		"jwks_url":               "http://fake.anotherexample.com",
		"bound_issuer":           "http://vault.example.com/",
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
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(resp.Error().Error(), "exactly one of") {
		t.Fatalf("got unexpected error: %v", resp.Error())
	}

	// remove oidc_discovery_url, but this still leaves too many
	delete(data, "oidc_discovery_url")

	req = &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      data,
	}
	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if resp == nil || !resp.IsError() {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(resp.Error().Error(), "exactly one of") {
		t.Fatalf("got unexpected error: %v", resp.Error())
	}

	// remove jwks_url so the config is now valid
	delete(data, "jwks_url")

	req = &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      authSourceTestPath,
		Storage:   storage,
		Data:      data,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	pubkey, err := certutil.ParsePublicKeyPEM([]byte(testJWTPubKey))
	if err != nil {
		t.Fatal(err)
	}

	expected := &authSource{
		ParsedJWTPubKeys:     []crypto.PublicKey{pubkey},
		JWTValidationPubKeys: []string{testJWTPubKey},
		JWTSupportedAlgs:     []string{},
		OIDCResponseTypes:    []string{},
		BoundIssuer:          "http://vault.example.com/",
		NamespaceInState:     true,
	}

	conf, err := b.(*flantIamAuthBackend).authSourceConfig(context.Background(), NewPrefixStorage("source/", storage), authSourceTestName)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expected, conf) {
		t.Fatalf("expected did not match actual: expected %#v\n got %#v\n", expected, conf)
	}
}

// func TestAuthSource_JWKS_Update(t *testing.T) {
//	b, storage := getBackend(t)
//
//	s := newOIDCProvider(t)
//	defer s.server.Close()
//
//	cert, err := s.getTLSCert()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	data := map[string]interface{}{
//		"jwks_url":               s.server.URL + "/certs",
//		"jwks_ca_pem":            cert,
//		"oidc_discovery_url":     "",
//		"oidc_discovery_ca_pem":  "",
//		"oidc_client_id":         "",
//		"oidc_response_mode":     "form_post",
//		"oidc_response_types":    []string{},
//		"default_role":           "",
//		"jwt_validation_pubkeys": []string{},
//		"jwt_supported_algs":     []string{},
//		"bound_issuer":           "",
//		"provider_config":        map[string]interface{}{},
//		"namespace_in_state":     false,
//	}
//
//	req := &logical.Request{
//		Operation: logical.UpdateOperation,
//		Path:      authSourceTestPath,
//		Storage:   storage,
//		Data:      data,
//	}
//
//	resp, err := b.HandleRequest(context.Background(), req)
//	if err != nil || (resp != nil && resp.IsError()) {
//		t.Fatalf("err:%s resp:%#v\n", err, resp)
//	}
//
//	req = &logical.Request{
//		Operation: logical.ReadOperation,
//		Path:      authSourceTestPath,
//		Storage:   storage,
//		Data:      nil,
//	}
//
//	resp, err = b.HandleRequest(context.Background(), req)
//	if err != nil || (resp != nil && resp.IsError()) {
//		t.Fatalf("err:%s resp:%#v\n", err, resp)
//	}
//
//	if diff := deep.Equal(resp.Data, data); diff != nil {
//		t.Fatalf("Expected did not equal actual: %v", diff)
//	}
//}

// func TestAuthSource_JWKS_Update_Invalid(t *testing.T) {
//	b, storage := getBackend(t)
//
//	s := newOIDCProvider(t)
//	defer s.server.Close()
//
//	cert, err := s.getTLSCert()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	data := map[string]interface{}{
//		"jwks_url":               s.server.URL + "/certs_missing",
//		"jwks_ca_pem":            cert,
//		"oidc_discovery_url":     "",
//		"oidc_discovery_ca_pem":  "",
//		"oidc_client_id":         "",
//		"default_role":           "",
//		"jwt_validation_pubkeys": []string{},
//		"jwt_supported_algs":     []string{},
//		"bound_issuer":           "",
//	}
//
//	req := &logical.Request{
//		Operation: logical.UpdateOperation,
//		Path:      authSourceTestPath,
//		Storage:   storage,
//		Data:      data,
//	}
//
//	resp, err := b.HandleRequest(context.Background(), req)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if resp == nil || !resp.IsError() {
//		t.Fatal("expected error")
//	}
//	if !strings.Contains(resp.Error().Error(), "get keys failed") {
//		t.Fatalf("got unexpected error: %v", resp.Error())
//	}
//
//	data["jwks_url"] = s.server.URL + "/certs_invalid"
//
//	req = &logical.Request{
//		Operation: logical.UpdateOperation,
//		Path:      authSourceTestPath,
//		Storage:   storage,
//		Data:      data,
//	}
//
//	resp, err = b.HandleRequest(context.Background(), req)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if resp == nil || !resp.IsError() {
//		t.Fatal("expected error")
//	}
//	if !strings.Contains(resp.Error().Error(), "failed to decode keys") {
//		t.Fatalf("got unexpected error: %v", resp.Error())
//	}
//}
//
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
		"oidc_discovery_url":    "https://team-vault.auth0.com/",
		"oidc_discovery_ca_pem": oidcBadCACerts,
		"oidc_client_id":        "abc",
		"oidc_client_secret":    "def",
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

	expected := &authSource{
		JWTValidationPubKeys: []string{},
		JWTSupportedAlgs:     []string{},
		OIDCResponseTypes:    []string{},
		OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
		OIDCClientID:         "abc",
		OIDCClientSecret:     "def",
		NamespaceInState:     true,
	}

	conf, err := b.(*flantIamAuthBackend).authSourceConfig(context.Background(), NewPrefixStorage("source/", storage), authSourceTestName)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(expected, conf); diff != nil {
		t.Fatal(diff)
	}

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
		expected authSource
	}
	tests := map[string]testCase{
		"namespace_in_state not specified": {
			create: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
			},
		},
		"namespace_in_state true": {
			create: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
			},
		},
		"namespace_in_state false": {
			create: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
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

			conf, err := b.(*flantIamAuthBackend).authSourceConfig(context.Background(), NewPrefixStorage("source/", storage), authSourceTestName)
			assert.NoError(t, err)
			assert.Equal(t, &test.expected, conf)
		})
	}
}

func TestAuthSource_OIDC_Update_Namespace(t *testing.T) {
	type testCase struct {
		existing map[string]interface{}
		update   map[string]interface{}
		expected authSource
	}
	tests := map[string]testCase{
		"existing false, update to true": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
			},
		},
		"existing false, update something else": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"default_role":       "ui",
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				DefaultRole:          "ui",
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
			},
		},
		"existing true, update to false": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": false,
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     false,
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
			},
		},
		"existing true, update something else": {
			existing: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"namespace_in_state": true,
			},
			update: map[string]interface{}{
				"oidc_discovery_url": "https://team-vault.auth0.com/",
				"default_role":       "ui",
			},
			expected: authSource{
				OIDCDiscoveryURL:     "https://team-vault.auth0.com/",
				NamespaceInState:     true,
				DefaultRole:          "ui",
				OIDCResponseTypes:    []string{},
				JWTSupportedAlgs:     []string{},
				JWTValidationPubKeys: []string{},
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

			conf, err := b.(*flantIamAuthBackend).authSourceConfig(context.Background(), NewPrefixStorage("source/", storage), authSourceTestName)
			assert.NoError(t, err)
			assert.Equal(t, &test.expected, conf)
		})
	}
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
