package vault_client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

var NotSetConfError = fmt.Errorf("access configuration does not set")
var ClientNotInitError = fmt.Errorf("client not init")

type VaultClientController struct {
	clientLock sync.RWMutex
	apiClient  *api.Client

	storageFactory func(s logical.Storage) *accessConfigStorage
}

func NewVaultClientController() *VaultClientController {
	c := &VaultClientController{}

	c.storageFactory = func(s logical.Storage) *accessConfigStorage {
		return &accessConfigStorage{
			parent: s,
		}
	}

	return c
}

func (c *VaultClientController) Init(s logical.Storage) error {
	apiClient, err := c.ApiClient()
	if err == nil && apiClient != nil {
		return nil
	}

	ctx := context.Background()
	curConf, err := c.storageFactory(s).Get(ctx)
	if err != nil {
		return err
	}

	if curConf == nil {
		return NotSetConfError
	}

	apiClient, err = newApiClient(curConf)
	if err != nil {
		return err
	}

	accessClient := newAccessClient(apiClient, curConf)
	auth, err := accessClient.AppRole().Login()

	if err != nil {
		return err
	}

	apiClient.SetToken(auth.ClientToken)
	c.setApiClient(apiClient)

	return nil
}

func (c *VaultClientController) ApiClient() (*api.Client, error) {
	c.clientLock.RLock()
	defer c.clientLock.RUnlock()

	if c.apiClient != nil {
		return nil, ClientNotInitError
	}

	return c.apiClient, nil
}

func (c *VaultClientController) setApiClient(newClient *api.Client) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	c.apiClient = newClient
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, store *accessConfigStorage, curConf *vaultAccessConfig) error {
	apiClient, err := c.ApiClient()
	if errors.Is(err, ClientNotInitError) || apiClient == nil {
		apiClient, err = newApiClient(curConf)
		if err != nil {
			return err
		}

		defer c.setApiClient(apiClient)
	}

	// login in with new secret id in gen function
	err = genNewSecretId(ctx, apiClient, store, curConf)
	if err != nil {
		return err
	}

	return nil
}
