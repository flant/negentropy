package root

import (
	"fmt"

	"github.com/hashicorp/go-hclog"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ObjectHandler struct {
	entityRepo       *model.EntityRepo
	eaRepo           *model.EntityAliasRepo
	authSourceRepo   *repo.AuthSourceRepo
	multipassGenRepo *model.MultipassGenerationNumberRepository

	logger hclog.Logger
}

func NewObjectHandler(txn *io.MemoryStoreTxn, logger hclog.Logger) *ObjectHandler {
	return &ObjectHandler{
		entityRepo:       model.NewEntityRepo(txn),
		eaRepo:           model.NewEntityAliasRepo(txn),
		authSourceRepo:   repo.NewAuthSourceRepo(txn),
		multipassGenRepo: model.NewMultipassGenerationNumberRepository(txn),
		logger:           logger,
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

	err = h.authSourceRepo.Iter(true, func(source *model.AuthSource) (bool, error) {
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

	err = h.authSourceRepo.Iter(true, func(source *model.AuthSource) (bool, error) {
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

func (h *ObjectHandler) HandleMultipass(mp *iam.Multipass) error {
	l := h.logger

	l.Debug(fmt.Sprintf("Handle multipass %s", mp.UUID))
	genNum, err := h.multipassGenRepo.GetByID(mp.UUID)
	if err != nil {
		return err
	}

	if genNum != nil {
		l.Debug(fmt.Sprintf("Found multipass generation number %s. Skip create", mp.UUID))
		return nil
	}

	genNum = &model.MultipassGenerationNumber{
		UUID:             mp.UUID,
		GenerationNumber: 0,
	}

	l.Debug(fmt.Sprintf("Try to create generation number for multipass %s", mp.UUID))
	return h.multipassGenRepo.Create(genNum)
}

func (h *ObjectHandler) HandleDeleteMultipass(uuid string) error {
	h.logger.Debug(fmt.Sprintf("Handle delete multipass multipass %s. Try to delete multipass gen number", uuid))
	return h.multipassGenRepo.Delete(uuid)
}

func (h *ObjectHandler) deleteEntityWithAliases(uuid string) error {
	// begin delete entity aliases
	err := h.eaRepo.GetAllForUser(uuid, func(alias *model.EntityAlias) (bool, error) {
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
