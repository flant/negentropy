package vault_client

import (
	"time"
)

const storagePath = "configure_vault_access"

type vaultAccessConfig struct {
	ApiUrl            string        `json:"vault_api_url"`
	ApiHost           string        `json:"vault_api_host"`
	ApiCa             string        `json:"vault_api_ca"`
	RoleName          string        `json:"role_name"`
	SecretId          string        `json:"secret_id"`
	RoleId            string        `json:"role_id"`
	SecretIdTtl       time.Duration `json:"secret_id_ttl"`
	ApproleMountPoint string        `json:"approle_mount_point"`
	LastRenewTime     time.Time     `json:"last_renew_time"`
}
