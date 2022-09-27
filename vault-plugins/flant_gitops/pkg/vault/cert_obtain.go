package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

type CertAndKey struct {
	CertPem       string
	PrivateKeyPem string
}

// ObtainCertAndKey try got access to specific endpoint at vault
// negentropyVaultClient is a client which has access to 'vault-cert-auth/issue/cert-auth'
// at vault-cert-auth should be mounted 'pki' plugin
// at vault-cert-auth should be created role 'cert-auth', example:
// vault write  vault-cert-auth/roles/cert-auth allow_any_name='true' max_ttl='1h'
func ObtainCertAndKey(negentropyVaultClient *api.Client, commonName string) (*CertAndKey, error) {
	secret, err := negentropyVaultClient.Logical().Write("vault-cert-auth/issue/cert-auth", map[string]interface{}{"common_name": commonName})
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("empty secret or secret.data")
	}
	certPem := secret.Data["certificate"].(string)
	if certPem == "" {
		return nil, fmt.Errorf("certificate is empty, or not returned")
	}
	pkPem := secret.Data["private_key"].(string)
	if pkPem == "" {
		return nil, fmt.Errorf("private_key is empty, or not returned")
	}
	return &CertAndKey{
		CertPem:       certPem,
		PrivateKeyPem: pkPem,
	}, nil
}
