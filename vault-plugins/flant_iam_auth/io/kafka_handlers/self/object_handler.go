package self

import (
	"fmt"
	"github.com/hashicorp/go-hclog"

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

	logger hclog.Logger
}

func NewObjectHandler(memStore *io.MemoryStore, txn *io.MemoryStoreTxn, api *vault.VaultEntityDownstreamApi, logger hclog.Logger) *ObjectHandler {
	return &ObjectHandler{
		eaRepo:     model.NewEntityAliasRepo(txn),
		entityRepo: model.NewEntityRepo(txn),
		usersRepo:  iamrepos.NewUserRepository(txn),
		saRepo:     iamrepos.NewServiceAccountRepository(txn),
		memStore:   memStore,
		txn:        txn,
		downstream: api,
		logger:     logger,
	}
}

func (h *ObjectHandler) HandleAuthSource(source *model.AuthSource) error {
	l := h.logger
	l.Debug("Handle auth source", source.Name)
	err := h.usersRepo.Iter(func(user *iamrepos.User) (bool, error) {
		l.Debug("Create new ea mem object for user an source", user.FullIdentifier, source.Name)
		err := h.eaRepo.CreateForUser(user, source)
		if err != nil {
			l.Error("Cannot create ea mem object for user an source", user.FullIdentifier, source.Name, err)
			return false, err
		}

		l.Debug("Create new ea mem object for user an source", user.FullIdentifier)

		return true, nil
	})
	if err != nil {
		return nil
	}

	if !source.AllowForSA() {
		l.Error("Source not allow for SA skip", source.Name)
		return nil
	}

	return h.saRepo.Iter(func(account *iamrepos.ServiceAccount) (bool, error) {
		l.Debug("Create new ea mem object for SA and source", account.FullIdentifier, source.Name)
		err := h.eaRepo.CreateForSA(account, source)
		if err != nil {
			l.Error("Cannot create ea mem object for SA an source", account.FullIdentifier, source.Name, err)
			return false, nil
		}

		l.Debug("Created new ea mem object for SA and source", account.FullIdentifier, source.Name)
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
	l := h.logger

	l.Debug("Handle delete source", uuid)
	err := h.eaRepo.GetBySource(uuid, func(alias *model.EntityAlias) (bool, error) {
		l.Debug("Delete entity alias obj", alias.UUID, alias.Name)
		err := h.eaRepo.DeleteByID(alias.UUID)
		if err != nil {
			l.Debug("Can not delete entity alias obj", alias.UUID, alias.Name, err)
			return false, err
		}
		l.Debug("Deleted entity alias obj", alias.UUID, alias.Name)
		return true, nil
	})

	return err
}

func (h *ObjectHandler) DeletedEntity(id string) error {
	return h.processActions(h.downstream.ProcessDeleteEntity(h.memStore, h.txn, id))
}

func (h *ObjectHandler) DeletedEntityAlias(id string) error {
	actions, err := h.downstream.ProcessDeleteEntityAlias(h.memStore, h.txn, id)
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
