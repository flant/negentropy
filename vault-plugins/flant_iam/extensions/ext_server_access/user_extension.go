package ext_server_access

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"k8s.io/apimachinery/pkg/util/wait"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/txnwatchers"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func InitializeExtensionServerAccess(ctx context.Context, initRequest *logical.InitializationRequest,
	memStore *io.MemoryStore) error {
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
		ObjType: iam_model.UserType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			user := obj.(*iam_model.User)
			err := CreateExtension(user, sac)
			if err != nil {
				return err
			}

			log.L().Debug("finished user hook", "new", obj)

			return nil
		},
	})

	// TODO: refactor this bullshit
	memStore.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: iam_model.ProjectType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			groupRepo := iam_repo.NewGroupRepository(txn)
			projectRepo := iam_repo.NewProjectRepository(txn)

			project := obj.(*iam_model.Project)

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

			var groupToChange *iam_model.Group
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
		ObjType: iam_model.GroupType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newGroup, ok := obj.(*iam_model.Group)
			if !ok {
				return fmt.Errorf("need Group type, actually passed %T", obj)
			}
			repo := iam_repo.NewGroupRepository(txn)
			oldGroup, err := repo.GetByID(newGroup.UUID)
			if err == consts.ErrNotFound {
				err = nil
				oldGroup = &iam_model.Group{}
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
		ObjType: iam_model.RoleBindingType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newRoleBinding, ok := obj.(*iam_model.RoleBinding)
			if !ok {
				return fmt.Errorf("need RoleBinding type, actually passed %T", obj)
			}
			repo := iam_repo.NewRoleBindingRepository(txn)
			oldRoleBinding, err := repo.GetByID(newRoleBinding.UUID)
			if err == consts.ErrNotFound {
				err = nil
				oldRoleBinding = &iam_model.RoleBinding{}
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
		ObjType: iam_model.RoleType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			newRole, ok := obj.(*iam_model.Role)
			if !ok {
				return fmt.Errorf("need Role type, actually passed %T", obj)
			}
			repo := iam_repo.NewRoleRepository(txn)
			oldRole, err := repo.GetByID(newRole.Name)
			if err == consts.ErrNotFound {
				err = nil
				oldRole = &iam_model.Role{}
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
	sac *ServerAccessConfig, accounts map[iam_model.ServiceAccountUUID]struct{}) error {
	// TODO implement for serviceAccounts
	return nil
}

func checkAndCreateUserExtension(txn *io.MemoryStoreTxn, sac *ServerAccessConfig,
	users map[iam_model.UserUUID]struct{}, vaultStore logical.Storage) error {
	repoUser := iam_repo.NewUserRepository(txn)

	for userUUID := range users {
		user, err := repoUser.GetByID(userUUID)
		if err != nil {
			return fmt.Errorf("checkAndCreateUserExtension: %w", err)
		}
		if _, isSet := user.Extensions[consts.OriginServerAccess]; !isSet {
			err = CreateExtension(user, sac)
			if err != nil {
				return fmt.Errorf("creating user extension: %w", err)
			}
		}
	}
	return nil
}

func CreateExtension(user *iam_model.User, sac *ServerAccessConfig) error {
	if user.Extensions == nil {
		user.Extensions = map[consts.ObjectOrigin]*iam_model.Extension{}
	}

	if _, ok := user.Extensions[consts.OriginServerAccess]; ok {
		return nil
	}

	randomSeed, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return nil
	}

	randomSalt, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return nil
	}

	lastUID := sac.LastAllocatedUID
	currentUID := lastUID + 1

	user.Extensions[consts.OriginServerAccess] = &iam_model.Extension{
		Origin:    consts.OriginServerAccess,
		OwnerType: iam_model.ExtensionOwnerTypeUser,
		OwnerUUID: user.ObjId(),
		Attributes: map[string]interface{}{
			"UID": currentUID,
			"passwords": []ext_model.UserServerPassword{
				{
					Seed:      randomSeed,
					Salt:      randomSalt,
					ValidTill: time.Time{},
				},
			},
		},
		SensitiveAttributes: nil, // TODO: ?
	}
	sac.LastAllocatedUID = currentUID

	return nil
}

func RevealPassword(tx *io.MemoryStoreTxn, sac *ServerAccessConfig,
	userUUID, serverUUID string) (string, error) {
	user, err := iam_repo.NewUserRepository(tx).GetByID(userUUID)
	if err != nil {
		return "", err
	}

	randomSeed, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return "", err
	}

	randomSalt, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return "", err
	}

	passwordsRaw := user.Extensions[consts.OriginServerAccess].Attributes["passwords"]
	passwords := passwordsRaw.([]ext_model.UserServerPassword)

	passwords = garbageCollectPasswords(passwords, randomSeed, randomSalt,
		sac.ExpirePasswordSeedAfterRevealIn, sac.DeleteExpiredPasswordSeedsAfter)

	freshPass, err := returnFreshPassword(passwords)
	if err != nil {
		return "", err
	}

	sha512Hash := sha512.New()
	_, err = sha512Hash.Write(append([]byte(serverUUID), freshPass.Seed...))
	retPass := hex.EncodeToString(sha512Hash.Sum(nil))

	return retPass[:11], nil
}

var NoValidPasswords = errors.New("no valid Password found in User extension")

func returnFreshPassword(usps []ext_model.UserServerPassword) (ext_model.UserServerPassword, error) {
	if len(usps) == 0 {
		return ext_model.UserServerPassword{}, errors.New("no User password found")
	}

	sort.Slice(usps, func(i, j int) bool {
		return usps[i].ValidTill.Before(usps[j].ValidTill) // TODO: should iterate from freshest. check!!!
	})

	return usps[0], NoValidPasswords
}

func garbageCollectPasswords(usps []ext_model.UserServerPassword, seed, salt []byte,
	expirePasswordSeedAfterRevealIn, deleteAfter time.Duration) (ret []ext_model.UserServerPassword) {
	var (
		currentTime                            = time.Now()
		expirePasswordSeedAfterTimestamp       = currentTime.Add(expirePasswordSeedAfterRevealIn)
		expirePasswordSeedAfterTimestampHalved = currentTime.Add(expirePasswordSeedAfterRevealIn / 2)
		deleteAfterTimestamp                   = currentTime.Add(deleteAfter)
	)

	if !usps[len(usps)-1].ValidTill.After(expirePasswordSeedAfterTimestampHalved) {
		usps[len(usps)-1].ValidTill = time.Time{}
		usps = append(usps, ext_model.UserServerPassword{
			Seed:      seed,
			Salt:      salt,
			ValidTill: expirePasswordSeedAfterTimestamp,
		})
	}

	for _, usp := range usps {
		if !usp.ValidTill.Before(deleteAfterTimestamp) {
			ret = append(ret, usp)
		}
	}

	return
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
