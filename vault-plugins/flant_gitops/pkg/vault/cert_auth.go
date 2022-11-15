package vault

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
)

// ApiClientPreparedForAuthorizationByCert returns vault api client authorized by passed vaultRootCaCert, clientCert and clientKey
// vaultRootCaCert is a certificate which is a root in https vault certificate
// clientCert ia a certificate signed by CA which is passed to auth/cert as a tructed CA
// clientKey is a private pair of clientCert
// vaultAddress: example: https://localhost:8300
func ApiClientPreparedForAuthorizationByCert(vaultAddress string, vaultRootCaCert string, clientCert string, clientKey string) (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = vaultAddress
	httpClient, err := buildClient(vaultRootCaCert, clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed building https client: %w", err)
	}

	cfg.HttpClient = httpClient
	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed building vault api client: %w", err)
	}
	return cl, nil
}

// AuthorizeByCert try login and check and returns secret
// preparedClient is an api client prepared for cert login to https vault server
func AuthorizeByCert(preperedClient *api.Client) (*api.Secret, error) {
	secret, err := preperedClient.Logical().Write("auth/cert/login", nil)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, fmt.Errorf("empty secret")
	}
	if secret.Auth == nil {
		return nil, fmt.Errorf("empty secret.auth")
	}
	return secret, nil
}

func buildClient(vaultRootCaCert string, clientCert string, clientKey string) (*http.Client, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(vaultRootCaCert))

	publicCertificateBlock, err := parsePublicCertificate(clientCert)
	if err != nil {
		return nil, err
	}
	privateKey, err := parsePrivateKey(clientKey)
	if err != nil {
		return nil, err
	}

	certificate := tls.Certificate{
		Certificate: [][]byte{publicCertificateBlock.Bytes},
		PrivateKey:  privateKey,
	}

	c := http.DefaultClient
	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS12,
		},
	}
	return c, nil
}

func parsePrivateKey(rawPem string) (crypto.PrivateKey, error) {
	block, rest := pem.Decode([]byte(rawPem))
	if len(rest) > 0 {
		return nil, fmt.Errorf("wrong raw block, expected only one block, got not zero 'rest' after pem.Decode")
	}
	if block == nil {
		return nil, fmt.Errorf("wrong raw block, got nil block after pem.Decode")
	}
	der := block.Bytes
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("failed to parse private key")
}

func parsePublicCertificate(rawPem string) (*pem.Block, error) {
	block, rest := pem.Decode([]byte(rawPem))
	if len(rest) > 0 {
		return nil, fmt.Errorf("wrong raw block, expected only one block, got not zero 'rest' after pem.Decode")
	}
	if block == nil {
		return nil, fmt.Errorf("wrong raw block, got nil block after pem.Decode")
	}
	return block, nil
}

// BuildVaultsBase64Env returns prepared value like: '[{"name":"vault-root-1", "url":"http://127.0.0.1:8200/", "token":"hvs.KqN6TSCyx9CFbTxCVCORwASN"}]'
// encoded with Base64
// if some vault doesn't response - one warnings is added to second returned value,
func BuildVaultsBase64Env(ctx context.Context, storage logical.Storage, client *api.Client, logger log.Logger) (string, []string, error) {
	vaults, warnings, err := buildVaultsEnv(ctx, storage, client)
	if err != nil {
		return vaults, warnings, err
	}
	logger.Debug("built vaults json", "vaults", vaults)
	vaultsBase64Json := b64.StdEncoding.EncodeToString([]byte(vaults))
	return vaultsBase64Json, warnings, nil
}

// BuildVaultsEnv returns prepared value like: '[{"name":"vault-root-1", "url":"http://127.0.0.1:8200/", "token":"hvs.KqN6TSCyx9CFbTxCVCORwASN"}]'
// if some vault doesn't response - one warnings is added to second returned value,
func buildVaultsEnv(ctx context.Context, storage logical.Storage, client *api.Client) (string, []string, error) {
	certAndKey, err := ObtainCertAndKey(client, "conf")
	if err != nil {
		return "", nil, err
	}
	vaultsConfig, err := getConfiguration(ctx, storage)
	if err != nil {
		return "", nil, err
	}
	var vaults []vaultWithToken
	var warnings []string
	for _, v := range vaultsConfig.Vaults {
		err = backoff.Retry(func() error {
			vault, err := buildVaultWithToken(v, *certAndKey)
			if err != nil {
				return err
			}
			vaults = append(vaults, *vault)
			return nil
		}, sharedio.TwoMinutesBackoff())
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("collecting token for vault: %s, at %s returns error: %s", v.VaultName, v.VaultUrl, err.Error()))
		}
	}
	data, err := json.Marshal(vaults)
	if err != nil {
		return "", nil, err
	}
	return string(data), warnings, nil
}

type vaultWithToken struct {
	VaultConfiguration
	VaultToken string `structs:"token" json:"token"`
}

// buildVaultWithToken obtain vault by cert authorization and returns token inside vaultWithToken
func buildVaultWithToken(v VaultConfiguration, certAndKey CertAndKey) (*vaultWithToken, error) {
	preparedClient, err := ApiClientPreparedForAuthorizationByCert(v.VaultUrl, v.CaCert, certAndKey.CertPem, certAndKey.PrivateKeyPem)
	if err != nil {
		return nil, err
	}
	secret, err := AuthorizeByCert(preparedClient)
	return &vaultWithToken{
		VaultConfiguration: v,
		VaultToken:         secret.Auth.ClientToken,
	}, nil
}
