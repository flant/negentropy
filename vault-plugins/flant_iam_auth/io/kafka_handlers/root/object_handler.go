package root

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/hashicorp/go-hclog"
)

type ObjectHandler struct {
	entityRepo     *model.EntityRepo
	eaRepo         *model.EntityAliasRepo
	authSourceRepo *repo.AuthSourceRepo

	logger hclog.Logger
}

func NewObjectHandler(txn *io.MemoryStoreTxn, logger hclog.Logger) *ObjectHandler {
	return &ObjectHandler{
		entityRepo:     model.NewEntityRepo(txn),
		eaRepo:         model.NewEntityAliasRepo(txn),
		authSourceRepo: repo.NewAuthSourceRepo(txn),
		logger:         logger,
	}
}

func (h *ObjectHandler) HandleUser(user *iam.User) error {
	l := h.logger

	l.Debug("Handle new user. Create entity object", user.FullIdentifier)
	err := h.entityRepo.CreateForUser(user)
	if err != nil {
		return err
	}
	l.Debug("Entity object created for user", user.FullIdentifier)

	err = h.authSourceRepo.Iter(func(source *model.AuthSource) (bool, error) {
		l.Debug("Create entity alias for user and source", user.FullIdentifier, source.Name)
		err := h.eaRepo.CreateForUser(user, source)
		if err != nil {
			return false, err
		}

		l.Debug("Entity alias for user and source created", user.FullIdentifier, source.Name)
		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (h *ObjectHandler) HandleServiceAccount(sa *iam.ServiceAccount) error {
	err := h.entityRepo.CreateForSA(sa)
	if err != nil {
		return err
	}

	err = h.authSourceRepo.Iter(func(source *model.AuthSource) (bool, error) {
		err := h.eaRepo.CreateForSA(sa, source)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	return nil
}

func (h *ObjectHandler) HandleDeleteUser(uuid string) error {
	return h.deleteEntityWithAliases(uuid)
}

func (h *ObjectHandler) HandleDeleteServiceAccount(uuid string) error {
	return h.deleteEntityWithAliases(uuid)
}

func (h *ObjectHandler) deleteEntityWithAliases(uuid string) error {
	// begin delete entity aliases
	err := h.eaRepo.GetForUser(uuid, func(alias *model.EntityAlias) (bool, error) {
		err := h.eaRepo.DeleteByID(alias.UUID)
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return h.entityRepo.DeleteForUser(uuid)
}
