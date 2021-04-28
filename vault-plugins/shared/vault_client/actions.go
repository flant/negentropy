package vault_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
)

func newApiClient(accessConf *vaultAccessConfig) (*api.Client, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(accessConf.ApiCa))

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	httpClient := api.DefaultConfig().HttpClient
	httpClient.Transport = transport
	clientConf := &api.Config{
		Address:    accessConf.ApiUrl,
		HttpClient: httpClient,
	}

	client, err := api.NewClient(clientConf)
	if err != nil {
		return nil, err
	}

	client.AddHeader("host", accessConf.ApiHost)

	return client, nil
}

func genNewSecretId(ctx context.Context, apiClient *api.Client, store *accessConfigStorage, accessConf *vaultAccessConfig, logger log.Logger) error {
	// login with current secret id if no login current
	if apiClient.Token() == "" {
		err := loginAndSetToken(apiClient, accessConf, logger)
		if err != nil {
			return err
		}
	}

	// generate ne w secret id
	appRoleCli := newAccessClient(apiClient, accessConf, logger).AppRole()

	newSecretId, err := appRoleCli.GenNewSecretId()
	if err != nil {
		return err
	}

	// save new secret id in store
	oldSecretId := accessConf.SecretId
	accessConf.SecretId = newSecretId
	accessConf.LastRenewTime = time.Now()

	err = store.Put(ctx, accessConf)
	if err != nil {
		return err
	}

	err = apiClient.Auth().Token().RevokeSelf("" /*ignored*/)
	if err != nil {
		logger.Warn(fmt.Sprintf("does not revoke old access token %v", err))
		// no return error. token always revoked later
	}

	// before delete old secret we need login with new secret and set new token
	err = loginAndSetToken(apiClient, accessConf, logger)
	if err != nil {
		return err
	}

	// delete old secret from vault
	if oldSecretId != "" {
		err = appRoleCli.DeleteSecretId(oldSecretId)
		if err != nil {
			return err
		}
	}

	return nil
}

func loginAndSetToken(apiClient *api.Client, curConf *vaultAccessConfig, logger log.Logger) error {
	appRoleCli := newAccessClient(apiClient, curConf, logger).AppRole()

	loginRes, err := appRoleCli.Login()
	if err != nil {
		return err
	}

	apiClient.SetToken(loginRes.ClientToken)
	return nil
}

func prolongAccessToken(apiClient *api.Client, increment int) error {
	var err error
	for i := 0; i < 5; i++ {
		_, err = apiClient.Auth().Token().Renew(apiClient.Token(), increment)

		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		return nil
	}

	return err
}
