package vault

import (
	"fmt"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/vault-plugins/shared/client"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type MountAccessorGetter struct {
	vaultClientProvider client.VaultClientController
	path                string

	mutex    sync.Mutex
	accessor string
}

func NewMountAccessorGetter(vaultClientProvider client.VaultClientController, path string) *MountAccessorGetter {
	return &MountAccessorGetter{
		path:                path,
		vaultClientProvider: vaultClientProvider,
	}
}

// MountAccessor getting mount accessor
// with backoff
func (a *MountAccessorGetter) MountAccessor() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.accessor != "" {
		return a.accessor, nil
	}

	var authLists map[string]*api.AuthMount
	err := backoff.Retry(func() error {
		var err error
		client, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return nil
		}
		authLists, err = client.Sys().ListAuth()
		return err
	}, io.FiveSecondsBackoff())
	if err != nil {
		return "", err
	}

	mount, ok := authLists[a.path]
	if !ok {
		hasMounts := make([]string, 0)
		for k := range authLists {
			hasMounts = append(hasMounts, k)
		}
		return "", fmt.Errorf("not found auth mount %v, has %v", a.path, hasMounts)
	}

	a.accessor = mount.Accessor

	return a.accessor, nil
}
