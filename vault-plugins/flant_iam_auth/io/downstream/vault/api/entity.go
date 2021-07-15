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
		_, err := a.clientApi.Logical().Write("/identity/entity", map[string]interface{}{
			"name": name,
		})
		return err
	}
	return a.callOp(op)
}

func (a *EntityAPI) DeleteByName(name string) error {
	path := a.entityPath(name)
	op := func() error {
		_, err := a.clientApi.Logical().Delete(path)
		return err
	}
	return a.callOp(op)
}

func (a *EntityAPI) GetID(name string) (string, error) {
	var resp *api.Secret
	var err error

	path := a.entityPath(name)
	op := func() error {
		resp, err = a.clientApi.Logical().Read(path)
		if resp == nil {
			return fmt.Errorf("empty response in op")
		}
		return err
	}

	err = a.callOp(op)

	if err != nil {
		return "", err
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
	var err error

	path := a.entityPath(name)
	op := func() error {
		resp, err = a.clientApi.Logical().Read(path)
		return err
	}

	err = a.callOp(op)

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
