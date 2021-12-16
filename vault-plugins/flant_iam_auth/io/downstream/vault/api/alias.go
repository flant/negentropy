package api

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

type AliasAPI struct {
	*IdentityAPI
}

func (a *AliasAPI) Create(name string, entityID string, accessor string) error {
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		_, err = vaultClient.Logical().Write("/identity/entity-alias", map[string]interface{}{
			"name":           name,
			"canonical_id":   entityID,
			"mount_accessor": accessor,
		})
		return err
	}

	return a.callOp(op)
}

func (a *AliasAPI) DeleteByName(name string, accessor string) error {
	aliasId, err := a.FindAliasIDByName(name, accessor)
	if err != nil {
		return err
	}
	if aliasId == "" {
		return nil
	}

	return a.DeleteByID(aliasId)
}

func (a *AliasAPI) DeleteByID(id string) error {
	path := a.idPath(id)
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

func (a *AliasAPI) FindAliasIDByName(name string, accessor string) (string, error) {
	var resp *api.Secret
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		resp, err = vaultClient.Logical().Write("/identity/lookup/entity", map[string]interface{}{
			"alias_name":           name,
			"alias_mount_accessor": accessor,
		})
		return err
	}

	err := a.callOp(op)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("nil response")
	}

	aliasesRaw, ok := resp.Data["aliases"]
	aliases, okCast := aliasesRaw.([]interface{})
	if !ok || !okCast {
		return "", fmt.Errorf("cannot find aliases in response or don't cast it lookup entityAlias %s", name)
	}

	for _, aliasRaw := range aliases {
		alias, okCast := aliasRaw.(map[string]interface{})
		if !okCast {
			return "", fmt.Errorf("cannot find aliases in response or don't cast it lookup entityAlias %s", name)
		}
		nameRaw, ok := alias["name"]
		nameAlias, okCast := nameRaw.(string)
		if !ok || !okCast {
			return "", fmt.Errorf("cannot get alias name in response or don't cast it lookup entityAlias %s", name)
		}

		if nameAlias != name {
			continue
		}

		idRaw, ok := alias["id"]
		id, okCast := idRaw.(string)
		if !ok || !okCast {
			return "", fmt.Errorf("cannot get alias id in response or don't cast it lookup entityAlias %s", name)
		}

		return id, nil
	}

	return "", fmt.Errorf("cannot find entity alias %s", name)
}

func (a *AliasAPI) FindAliasByName(name string, accessor string) (map[string]interface{}, error) {
	var resp *api.Secret
	op := func() error {
		vaultClient, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		resp, err = vaultClient.Logical().Write("/identity/lookup/entity", map[string]interface{}{
			"alias_name":           name,
			"alias_mount_accessor": accessor,
		})
		return err
	}

	err := a.callOp(op)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	aliasesRaw, ok := resp.Data["aliases"]
	aliases, okCast := aliasesRaw.([]interface{})
	if !ok || !okCast {
		return nil, fmt.Errorf("cannot find aliases in response or don't cast it lookup entityAlias %s", name)
	}

	for _, aliasRaw := range aliases {
		alias, okCast := aliasRaw.(map[string]interface{})
		if !okCast {
			return nil, fmt.Errorf("cannot find aliases in response or don't cast it lookup entityAlias %s", name)
		}
		nameRaw, ok := alias["name"]
		nameAlias, okCast := nameRaw.(string)
		if !ok || !okCast {
			return nil, fmt.Errorf("cannot get alias name in response or don't cast it lookup entityAlias %s", name)
		}

		if nameAlias == name {
			return alias, nil
		}

	}

	return nil, fmt.Errorf("cannot find entity alias %s", name)
}

func (a *AliasAPI) idPath(id string) string {
	return fmt.Sprintf("/identity/entity-alias/id/%s", id)
}
