package vault_client

import (
	"context"
	utils "github.com/flant/negentropy/vault-plugins/shared/vault_backent_utils"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"net/url"
)

func PathConfigure(c *VaultClientController) *framework.Path {
	return &framework.Path{
		Pattern: `configure_vault_access$`,

		Fields: map[string]*framework.FieldSchema{
			"vault_api_url": {
				Type:        framework.TypeString,
				Description: `Url for connect to vault api`,
				Required:    true,
			},
			"vault_api_host": {
				Type:        framework.TypeString,
				Description: `Connection host. Uses as "Host" header in vault client`,
				Required:    true,
			},
			"vault_api_ca": {
				Type:        framework.TypeString,
				Description: "Vault CA cert using for TLS verification. In PEM format",
				Default:     "limbo",
				Required:    true,
			},
			"approle_mount_point": {
				Type:        framework.TypeString,
				Description: "Approle mount point for getting new token and renew it",
				Required:    true,
			},
			"role_name": {
				Type:        framework.TypeString,
				Description: "Role name to vault access",
				Required:    true,
			},
			"role_id": {
				Type:        framework.TypeString,
				Description: "Role id for approle",
				Required:    true,
			},
			"secret_id": {
				Type:        framework.TypeString,
				Description: "Secret to access approle",
				Required:    true,
			},
			"secret_id_ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Secret id time to life. Min 120s (2 minutes)",
				Required:    true,
			},
			"token_ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Token id time to life. Min 20s",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: c.handleConfigureVaultAccess,
				Summary:  configureVaultAccessSynopsis,
			},

			logical.AliasLookaheadOperation: &framework.PathOperation{
				Callback: c.handleConfigureVaultAccess,
			},
		},

		HelpSynopsis: configureVaultAccessSynopsis,
	}
}

func (c *VaultClientController) handleConfigureVaultAccess(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	config := &vaultAccessConfig{}
	var errResp *logical.Response

	config.ApiUrl, errResp = utils.NotEmptyStringParam(d, "vault_api_url")
	if errResp != nil {
		return errResp, nil
	}
	_, err := url.ParseRequestURI(config.ApiUrl)
	if err != nil {
		return logical.ErrorResponse("incorrect vault_api_url"), nil
	}

	config.ApiHost, errResp = utils.NotEmptyStringParam(d, "vault_api_host")
	if errResp != nil {
		return errResp, nil
	}

	//config.ApiCa, errResp = utils.NotEmptyStringParam(d, "vault_api_ca")
	//if errResp != nil {
	//	return errResp, nil
	//}
	//validPem, _ := pem.Decode([]byte(config.ApiCa))
	//if validPem == nil {
	//	return logical.ErrorResponse("incorrect vault_api_ca"), nil
	//}

	config.RoleName, errResp = utils.NotEmptyStringParam(d, "role_name")
	if errResp != nil {
		return errResp, nil
	}

	config.RoleId, errResp = utils.NotEmptyStringParam(d, "role_id")
	if errResp != nil {
		return errResp, nil
	}

	config.SecretId, errResp = utils.NotEmptyStringParam(d, "secret_id")
	if errResp != nil {
		return errResp, nil
	}

	config.SecretIdTtlSec, errResp = utils.DurationSecParam(d, "secret_id_ttl", 120)
	if errResp != nil {
		return errResp, nil
	}

	config.TokenTtlSec, errResp = utils.DurationSecParam(d, "token_ttl", 90)
	if errResp != nil {
		return errResp, nil
	}

	config.ApproleMountPoint, errResp = utils.NotEmptyStringParam(d, "approle_mount_point")
	if errResp != nil {
		return errResp, nil
	}

	err = c.setAccessConfig(ctx, c.storageFactory(req.Storage), config)
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

const (
	configureVaultAccessSynopsis = `
Configure vault access to itself
`
)
