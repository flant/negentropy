package vault_client

import (
	"context"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
)

func newApiClient(accessConf *vaultAccessConfig) (*api.Client, error) {
	tlsConf := &api.TLSConfig{
		CACert: accessConf.ApiCa,
	}
	clientConf := &api.Config{
		Address: accessConf.ApiUrl,
	}

	err := clientConf.ConfigureTLS(tlsConf)
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient(clientConf)
	if err != nil {
		return nil, err
	}

	client.AddHeader("host", accessConf.ApiHost)

	return client, nil
}

func genNewSecretId(ctx context.Context, apiClient *api.Client, store *accessConfigStorage, accessConf *vaultAccessConfig, logger log.Logger) error {
	// login with current secret id
	err := loginAndSetToken(apiClient, accessConf, logger)
	if err != nil {
		return err
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
