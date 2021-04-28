package client

import (
	"math"
	"time"
)

const storagePath = "configure_vault_access"

type vaultAccessConfig struct {
	APIURL            string        `json:"vault_api_url"`
	APIHost           string        `json:"vault_api_host"`
	APICa             string        `json:"vault_api_ca"`
	RoleName          string        `json:"role_name"`
	SecretID          string        `json:"secret_id"`
	RoleID            string        `json:"role_id"`
	SecretIDTTTLSec   time.Duration `json:"secret_id_ttl"`
	ApproleMountPoint string        `json:"approle_mount_point"`
	LastRenewTime     time.Time     `json:"last_renew_time"`
}

func (c *vaultAccessConfig) IsNeedToRenewSecretID(now time.Time) (bool, int) {

	if c.LastRenewTime.IsZero() {
		return true, 0
	}

	limit := math.Ceil(float64(c.SecretIDTTTLSec) * 0.8)
	diff := now.Sub(c.LastRenewTime).Seconds()

	return diff > limit, int(limit) - int(diff)
}
