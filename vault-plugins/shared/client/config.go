package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

const storagePath = "configure_vault_access"

type VaultApiConf struct {
	APIURL  string `json:"vault_api_url"`
	APIHost string `json:"vault_api_host"`
	CaCert  string `json:"vault_cacert"`
}

type vaultAccessConfig struct {
	VaultApiConf
	RoleName          string        `json:"role_name"`
	SecretID          string        `json:"secret_id"`
	RoleID            string        `json:"role_id"`
	SecretIDTTTLSec   time.Duration `json:"secret_id_ttl"`
	ApproleMountPoint string        `json:"approle_mount_point"`
	LastRenewTime     time.Time     `json:"last_renew_time"`
}

func (c *vaultAccessConfig) IsNeedToRenewSecretID(now time.Time) (bool, int) {
	if c.LastRenewTime.IsZero() {
		return true, 0
	}

	limit := math.Ceil(float64(c.SecretIDTTTLSec) * 0.1) // needs to have long live SecretID
	diff := now.Sub(c.LastRenewTime).Seconds()

	return diff > limit, int(limit) - int(diff)
}

// Preferable true if c should be used to configure client early configured with oldCfg
func (c *vaultAccessConfig) Preferable(oldCfg *vaultAccessConfig) bool {
	if c == nil {
		return false
	}
	if oldCfg == nil {
		return true
	}
	if *c != *oldCfg {
		return true
	}
	return false
}

func (c *vaultAccessConfig) newAPIClient() (*api.Client, error) {
	httpClient := api.DefaultConfig().HttpClient

	if strings.HasPrefix(c.APIURL, "https://127.0.0.1:") ||
		strings.HasPrefix(c.APIURL, "https://localhost:") {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // not_strictly_required_for_local_access
		}
		httpClient.Transport = &http.Transport{
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	} else if c.CaCert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(c.CaCert))
		// Setup HTTPS client
		tlsConfig := &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
		transport := &http.Transport{
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		httpClient.Transport = transport
	}

	clientConf := &api.Config{
		Address:    c.APIURL,
		HttpClient: httpClient,
	}

	client, err := api.NewClient(clientConf)
	if err != nil {
		return nil, err
	}

	client.AddHeader("host", c.APIHost)

	return client, nil
}

func getVaultClientConfig(ctx context.Context, storage logical.Storage) (*vaultAccessConfig, error) {
	raw, err := storage.Get(ctx, storagePath)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotSetConf
	}

	config := new(vaultAccessConfig)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func saveVaultClientConfig(ctx context.Context, storage logical.Storage, conf *vaultAccessConfig) error {
	entry, err := logical.StorageEntryJSON(storagePath, conf)
	if err != nil {
		return err
	}

	return storage.Put(ctx, entry)
}
