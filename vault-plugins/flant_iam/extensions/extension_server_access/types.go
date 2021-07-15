package extension_server_access

import "time"

type ServerAccessConfig struct {
	RolesForServers                  []string
	RoleForSSHAccess                 string
	DeleteExpiredPasswordSeedsAfter  time.Duration
	ExpirePasswordSeedAfterReveialIn time.Duration
	LastAllocatedUID                 int
}
