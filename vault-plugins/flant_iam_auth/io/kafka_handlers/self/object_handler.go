package self

import (
	"fmt"

	iamrepos "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ObjectHandler struct {
	eaRepo     *model.EntityAliasRepo
	entityRepo *model.EntityRepo
	usersRepo  *iamrepos.UserRepository
	saRepo     *iamrepos.ServiceAccountRepository
	downstream *vault.VaultEntityDownstreamApi
	memStore   *io.MemoryStore
	txn        *io.MemoryStoreTxn
}

func NewObjectHandler(memStore *io.MemoryStore, txn *io.MemoryStoreTxn, api *vault.VaultEntityDownstreamApi) *ObjectHandler {
	return &ObjectHandler{
		eaRepo:     model.NewEntityAliasRepo(txn),
		entityRepo: model.NewEntityRepo(txn),
		usersRepo:  iamrepos.NewUserRepository(txn),
		saRepo:     iamrepos.NewServiceAccountRepository(txn),
		memStore:   memStore,
		txn:        txn,
		downstream: api,
	}
}

func (h *ObjectHandler) HandleAuthSource(source *model.AuthSource) error {
	err := h.usersRepo.Iter(func(user *iamrepos.User) (bool, error) {
		err := h.eaRepo.CreateForUser(user, source)
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return nil
	}

	if !source.AllowForSA() {
		return nil
	}

	return h.saRepo.Iter(func(account *iamrepos.ServiceAccount) (bool, error) {
		err := h.eaRepo.CreateForSA(account, source)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
}

func (h *ObjectHandler) HandleEntity(entity *model.Entity) error {
	return h.processActions(h.downstream.ProcessEntity(h.memStore, h.txn, entity))
}

func (h *ObjectHandler) HandleEntityAlias(entityAlias *model.EntityAlias) error {
	return h.processActions(h.downstream.ProcessEntityAlias(h.memStore, h.txn, entityAlias))
}

func (h *ObjectHandler) DeletedAuthSource(uuid string) error {
	err := h.eaRepo.GetBySource(uuid, func(alias *model.EntityAlias) (bool, error) {
		err := h.eaRepo.DeleteByID(alias.UUID)
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return err
}

func (h *ObjectHandler) DeletedEntity(uuid string) error {
	entity, err := h.entityRepo.GetByID(uuid)
	if err != nil {
		return err
	}

	return h.processActions(h.downstream.ProcessDeleteEntity(h.memStore, h.txn, entity))
}

func (h *ObjectHandler) DeletedEntityAlias(uuid string) error {
	ea, err := h.eaRepo.GetById(uuid)
	if err != nil {
		return err
	}

	actions, err := h.downstream.ProcessDeleteEntityAlias(h.memStore, h.txn, ea)
	return h.processActions(actions, err)
}

func (h *ObjectHandler) processActions(actions []io.DownstreamAPIAction, err error) error {
	if err != nil {
		return err
	}

	if actions == nil {
		return nil
	}

	if len(actions) != 1 {
		return fmt.Errorf("incorrect actions for entity %v", len(actions))
	}

	return actions[0].Execute()
}