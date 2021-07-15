package extension_server_access

import (
	"strings"
	"time"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// FIXME: probably doesn't work

func RegisterServerAccessUserExtension(initialUID int,
	expireSeedAfterRevealIn time.Duration, deleteExpiredPasswordSeedsAfter time.Duration,
	storage *io.MemoryStore) error {

	storage.RegisterHook(io.ObjectHook{
		Events:  []io.HookEvent{io.HookEventInsert},
		ObjType: model.UserType,
		CallbackFn: func(txn *io.MemoryStoreTxn, _ io.HookEvent, obj interface{}) error {
			repo := model2.NewUserServerAccessRepository(txn, initialUID, expireSeedAfterRevealIn, deleteExpiredPasswordSeedsAfter)

			user := obj.(*model.User)

			err := repo.CreateExtension(user)
			if err != nil {
				return err
			}

			return nil
		},
	})

	// TODO: reconciliation?
	storage.RegisterHook(io.ObjectHook{
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

			for _, group := range groups {
				if !strings.HasPrefix(group.Identifier, "server/") {
					continue
				}

				projectRepo.GetByID(group.UUID)
			}

			return nil
		},
	})

	return nil
}
