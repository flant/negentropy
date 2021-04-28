package client

import (
	"context"
	"encoding/pem"
	"net/url"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	backendutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
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

	config.APIURL, errResp = backendutils.NotEmptyStringParam(d, "vault_api_url")
	if errResp != nil {
		return errResp, nil
	}
	_, err := url.ParseRequestURI(config.APIURL)
	if err != nil {
		return logical.ErrorResponse("incorrect vault_api_url"), nil
	}

	config.APIHost, errResp = backendutils.NotEmptyStringParam(d, "vault_api_host")
	if errResp != nil {
		return errResp, nil
	}

	config.APICa, errResp = backendutils.NotEmptyStringParam(d, "vault_api_ca")
	if errResp != nil {
		config.APICa = ""
	}
	if config.APICa != "" {
		validPem, _ := pem.Decode([]byte(config.APICa))
		if validPem == nil {
			return logical.ErrorResponse("incorrect vault_api_ca"), nil
		}
	}

	config.RoleName, errResp = backendutils.NotEmptyStringParam(d, "role_name")
	if errResp != nil {
		return errResp, nil
	}

	config.RoleID, errResp = backendutils.NotEmptyStringParam(d, "role_id")
	if errResp != nil {
		return errResp, nil
	}

	config.SecretID, errResp = backendutils.NotEmptyStringParam(d, "secret_id")
	if errResp != nil {
		return errResp, nil
	}

	config.SecretIDTTTLSec, errResp = backendutils.DurationSecParam(d, "secret_id_ttl", 120)
	if errResp != nil {
		return errResp, nil
	}

	config.ApproleMountPoint, errResp = backendutils.NotEmptyStringParam(d, "approle_mount_point")
	if errResp != nil {
		return errResp, nil
	}

	if strings.HasSuffix(config.ApproleMountPoint, "/") {
		config.ApproleMountPoint = strings.TrimSuffix(config.ApproleMountPoint, "/")
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
