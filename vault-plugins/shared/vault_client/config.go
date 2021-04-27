package vault_client

import (
	"math"
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
	SecretIdTtlSec    time.Duration `json:"secret_id_ttl"`
	ApproleMountPoint string        `json:"approle_mount_point"`
	LastRenewTime     time.Time     `json:"last_renew_time"`
}

func (c *vaultAccessConfig) IsNeedToRenewSecretId(now time.Time) (bool, int) {

	if c.LastRenewTime.IsZero() {
		return true, 0
	}

	limit := math.Ceil(float64(c.SecretIdTtlSec) * 0.8)
	diff := now.Sub(c.LastRenewTime).Seconds()

	return diff > limit, int(limit) - int(diff)
}
