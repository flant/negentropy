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

			groups, err := groupRepo.List(project.TenantUUID)
			if err != nil {
				return err
			}

			projects, err := projectRepo.List(project.TenantUUID)
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

	// update group hook
	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.GroupType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newGroup, ok := obj.(*model.Group)
			if !ok {
				return fmt.Errorf("need Group type, actually passed %T", obj)
			}
			repo := model.NewGroupRepository(txn)
			oldGroup, err := repo.GetByID(newGroup.UUID)
			if err == model.ErrNotFound {
				err = nil
				oldGroup = &model.Group{}
			}
			if err != nil {
				return fmt.Errorf("getting old group value: %w", err)
			}
			users, serverAccounts, err := txnwatchers.FindUsersAndSAsAffectedByPossibleRoleAddingOnGroupChange(txn,
				oldGroup, newGroup, sac.RoleForSSHAccess)
			if err != nil {
				return fmt.Errorf("check group transaction: %w", err)
			}
			err = checkAndCreateUserExtension(txn, sac, users, vaultStore)
			if err != nil {
				return fmt.Errorf("check and create user_extension: %w", err)
			}
			err = checkAndCreateServiceAccountExtension(txn, sac, serverAccounts)
			if err != nil {
				return fmt.Errorf("check and create servece_account_extension: %w", err)
			}
			return nil
		},
	})
	// update roleBinding hook
	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.RoleBindingType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newRoleBinding, ok := obj.(*model.RoleBinding)
			if !ok {
				return fmt.Errorf("need RoleBinding type, actually passed %T", obj)
			}
			repo := model.NewRoleBindingRepository(txn)
			oldRoleBinding, err := repo.GetByID(newRoleBinding.UUID)
			if err == model.ErrNotFound {
				err = nil
				oldRoleBinding = &model.RoleBinding{}
			}
			if err != nil {
				return fmt.Errorf("getting old roleBinding value: %w", err)
			}
			users, serverAccounts, err := txnwatchers.FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn,
				oldRoleBinding, newRoleBinding, sac.RoleForSSHAccess)
			if err != nil {
				return fmt.Errorf("check role_binding transaction: %w", err)
			}
			err = checkAndCreateUserExtension(txn, sac, users, vaultStore)
			if err != nil {
				return fmt.Errorf("check and create user_extension: %w", err)
			}
			err = checkAndCreateServiceAccountExtension(txn, sac, serverAccounts)
			if err != nil {
				return fmt.Errorf("check and create servece_account_extension: %w", err)
			}
			return nil
		},
	})

	// update role hook
	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.RoleType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newRole, ok := obj.(*model.Role)
			if !ok {
				return fmt.Errorf("need Role type, actually passed %T", obj)
			}
			repo := model.NewRoleRepository(txn)
			oldRole, err := repo.GetByID(newRole.Name)
			if err == model.ErrNotFound {
				err = nil
				oldRole = &model.Role{}
			}
			if err != nil {
				return fmt.Errorf("getting old roleBinding value: %w", err)
			}
			users, serverAccounts, err := txnwatchers.FindSubjectsAffectedByPossibleRoleAddingOnRoleChange(txn,
				oldRole, newRole, sac.RoleForSSHAccess)
			if err != nil {
				return fmt.Errorf("check role transaction: %w", err)
			}
			err = checkAndCreateUserExtension(txn, sac, users, vaultStore)
			if err != nil {
				return fmt.Errorf("check and create user_extension: %w", err)
			}
			err = checkAndCreateServiceAccountExtension(txn, sac, serverAccounts)
			if err != nil {
				return fmt.Errorf("check and create servece_account_extension: %w", err)
			}
			return nil
		},
	})
}

func checkAndCreateServiceAccountExtension(txn *io.MemoryStoreTxn,
	sac *ServerAccessConfig, accounts map[model.ServiceAccountUUID]struct{}) error {
	// TODO implement for serviceAccounts
	return nil
}

func checkAndCreateUserExtension(txn *io.MemoryStoreTxn, sac *ServerAccessConfig, users map[model.UserUUID]struct{}, vaultStore logical.Storage) error {
	repoUserExtension, err := ext_model.NewUserServerAccessRepository(txn,
		sac.LastAllocatedUID, sac.ExpirePasswordSeedAfterReveialIn, sac.DeleteExpiredPasswordSeedsAfter, vaultStore)
	if err != nil {
		return err
	}
	repoUser := model.NewUserRepository(txn)

	for userUUID := range users {
		user, err := repoUser.GetByID(userUUID)
		if err != nil {
			return fmt.Errorf("get user by id: %w", err)
		}
		if len(user.Extensions) >= 0 {
			if _, isSet := user.Extensions[model.OriginServerAccess]; !isSet {
				err = repoUserExtension.CreateExtension(user)
				if err != nil {
					return fmt.Errorf("creating user extension: %w", err)
				}
			}
		}
	}
	return nil
}
