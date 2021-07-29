package extension_server_access

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"k8s.io/apimachinery/pkg/util/wait"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func InitializeExtensionServerAccess(ctx context.Context, initRequest *logical.InitializationRequest, memStore *io.MemoryStore) error {
	storage := initRequest.Storage
	log.L().Debug("init server-access")

	config, err := liveConfig.GetServerAccessConfig(ctx, storage)
	if err != nil {
		log.L().Error("error get  server-access config", "err", err)
		return err
	}

	if config != nil {
		liveConfig.configured = true
	}

	go RegisterServerAccessUserExtension(context.Background(), storage, memStore)

	return nil
}

func RegisterServerAccessUserExtension(ctx context.Context, vaultStore logical.Storage, memStore *io.MemoryStore) {
	var sac *ServerAccessConfig

	_ = wait.PollImmediateInfinite(5*time.Second, func() (done bool, err error) {
		config, err := liveConfig.GetServerAccessConfig(ctx, vaultStore)
		if err != nil {
			log.L().Error("can't get current config from Vault Storage", "err", err)
			return false, nil
		}
		if config == nil {
			log.L().Info("server_access is not configured yet")

			return false, nil
		}

		sac = config
		log.L().Info("found server-access config", "config", config)

		return true, nil
	})

	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.UserType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			repo := model2.NewUserServerAccessRepository(txn, sac.LastAllocatedUID, sac.ExpirePasswordSeedAfterReveialIn, sac.DeleteExpiredPasswordSeedsAfter)

			user := obj.(*model.User)

			err := repo.CreateExtension(user)
			if err != nil {
				return err
			}

			log.L().Error("finished user hook", "new", obj)

			return nil
		},
	})

	// TODO: refactor this bullshit
	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.ProjectType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			groupRepo := model.NewGroupRepository(txn)
			projectRepo := model.NewProjectRepository(txn)

			project := obj.(*model.Project)

			groups, err := groupRepo.List(project.TenantUUID, false)
			if err != nil {
				return err
			}

			projects, err := projectRepo.List(project.TenantUUID, false)
			if err != nil {
				return err
			}

			projectIDSet := make(map[string]struct{}, len(projects))
			for _, project := range projects {
				projectIDSet[project.Identifier] = struct{}{}
			}

			var groupToChange *model.Group
			for _, group := range groups {
				if !strings.HasPrefix(group.Identifier, "servers/") {
					continue
				}

				splitGroupID := strings.Split(group.Identifier, "/")
				projectID := splitGroupID[1]

				if _, ok := projectIDSet[projectID]; !ok {
					continue
				}

				groupToChange = group
				groupToChange.Identifier = fmt.Sprintf("servers/%s", project.Identifier)

				break
			}

			if groupToChange == nil {
				return nil
			}

			return groupRepo.Update(groupToChange)
		},
	})
}
