package vault

import (
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type MountAccessorGetter struct {
	clientGetter func() (*api.Client, error)
	path         string

	mutex    sync.Mutex
	accessor string
}

func NewMountAccessorGetter(clientGetter io.BackoffClientGetter, path string) *MountAccessorGetter {
	return &MountAccessorGetter{
		path:         path,
		clientGetter: clientGetter,
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

	client, err := a.clientGetter()
	if err != nil {
		return "", err
	}

	var authLists map[string]*api.AuthMount
	backoffRequest := backoff.NewExponentialBackOff()
	backoffRequest.MaxElapsedTime = 5 * time.Second
	err = backoff.Retry(func() error {
		var err error
		authLists, err = client.Sys().ListAuth()
		return err
	}, backoffRequest)

	if err != nil {
		return "", err
	}

	mount, ok := authLists[a.path]
	if !ok {
		return "", fmt.Errorf("not found auth mount %v", a.path)
	}

	a.accessor = mount.Accessor

	return a.accessor, nil
}
