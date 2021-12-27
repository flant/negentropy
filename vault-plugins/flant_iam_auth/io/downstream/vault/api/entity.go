package api

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

type EntityAPI struct {
	*IdentityAPI
}

func (a *EntityAPI) Create(name string) error {
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		_, err = vaultClient.Logical().Write("/identity/entity", map[string]interface{}{
			"name": name,
		})
		return err
	}
	return a.callOp(op)
}

func (a *EntityAPI) DeleteByName(name string) error {
	path := a.entityPath(name)
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		_, err = vaultClient.Logical().Delete(path)
		return err
	}
	return a.callOp(op)
}

func (a *EntityAPI) GetID(name string) (string, error) {
	var resp *api.Secret

	path := a.entityPath(name)
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return fmt.Errorf("get APIClient:%w", err)
		}
		resp, err = vaultClient.Logical().Read(path)
		if err != nil {
			return fmt.Errorf("get entityID by name:%w", err)
		}
		if resp == nil {
			return fmt.Errorf("empty response in op")
		}
		return nil
	}

	err := a.callOp(op)
	if err != nil {
		return "", fmt.Errorf("callOp:%w", err)
	}

	if resp == nil {
		return "", fmt.Errorf("empty response")
	}

	idRaw, ok := resp.Data["id"]
	id, okCast := idRaw.(string)

	if !ok || !okCast {
		return "", fmt.Errorf("id not present in data or don't cast %s", name)
	}

	return id, nil
}

func (a *EntityAPI) GetByName(name string) (map[string]interface{}, error) {
	var resp *api.Secret

	path := a.entityPath(name)
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		resp, err = vaultClient.Logical().Read(path)
		return err
	}

	err := a.callOp(op)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	return resp.Data, nil
}

func (a *EntityAPI) entityPath(name string) string {
	return fmt.Sprintf("/identity/entity/name/%s", name)
}

func (a *EntityAPI) GetByID(entityID string) (map[string]interface{}, error) {
	var resp *api.Secret

	path := fmt.Sprintf("/identity/entity/id/%s", entityID)
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}

		resp, err = vaultClient.Logical().Read(path)
		return err
	}

	err := a.callOp(op)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	return resp.Data, nil
}
