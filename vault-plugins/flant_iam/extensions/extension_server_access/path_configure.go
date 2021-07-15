package extension_server_access

import (
	"context"
	"errors"
	"net/http"
	"path"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type serverConfigureBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func ServerConfigurePaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &serverConfigureBackend{
		Backend: b,
		storage: storage,
	}

	return bb.configurePaths()
}

func (b *serverConfigureBackend) configurePaths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("configure_extension", "server_access"),
			Fields: map[string]*framework.FieldSchema{
				"roles_for_servers": {
					Type:        framework.TypeStringSlice,
					Description: "List of roles assigned to newly created server Groups",
					Required:    true,
				},
				"role_for_ssh_access": {
					Type:        framework.TypeString,
					Description: "Role to use for SSH access",
					Required:    true,
				},
				"delete_expired_password_seeds_after": {
					Type:        framework.TypeDurationSecond,
					Description: "Duration after which expired password seed will be garbage collected",
					Required:    true,
				},
				"expire_password_seed_after_reveal_in": {
					Type:        framework.TypeDurationSecond,
					Description: "Duration after password reveal after which password seeds will be expired",
					Required:    true,
				},
				"last_allocated_uid": {
					Type:        framework.TypeInt,
					Description: "Last allocated POSIX UID",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfig(),
					Summary:  "Register server",
				},
			},
		},
	}
}

func (b *serverConfigureBackend) handleConfig() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		if !liveConfig.isConfigured() && data.Get("last_allocated_uid") == nil {
			return responseErr(req, errors.New(`"last_allocated_uid" not provided and config in storage is missing`))
		}

		var newServerAccessConfig ServerAccessConfig
		rawRolesForServers := data.Get("roles_for_servers")
		newServerAccessConfig.RolesForServers = rawRolesForServers.([]string)

		rawRoleForSSHAccess := data.Get("role_for_ssh_access")
		newServerAccessConfig.RoleForSSHAccess = rawRoleForSSHAccess.(string)

		rawDeleteExpiredPasswordSeedsAfter := data.Get("delete_expired_password_seeds_after")
		newServerAccessConfig.DeleteExpiredPasswordSeedsAfter = time.Duration(rawDeleteExpiredPasswordSeedsAfter.(int))

		rawExpirePasswordSeedAfterRevealIn := data.Get("expire_password_seed_after_reveal_in")
		newServerAccessConfig.ExpirePasswordSeedAfterReveialIn = time.Duration(rawExpirePasswordSeedAfterRevealIn.(int))

		if rawLastAllocatedUID, ok := data.GetOk("last_allocated_uid"); ok {
			newServerAccessConfig.LastAllocatedUID = rawLastAllocatedUID.(int)
		}

		err := liveConfig.SetServerAccessConfig(ctx, req.Storage, &newServerAccessConfig)
		if err != nil {
			return responseErr(req, err)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusOK)
	}
}
