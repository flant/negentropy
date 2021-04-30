package iam

import (
	"fmt"

	iamrepos "github.com/flant/negentropy/vault-plugins/flant_iam/backend"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ObjectHandler struct {
	eaRepo     *repo.EntityAliasRepo
	usersRepo  *iamrepos.UserRepository
	saRepo     *iamrepos.ServiceAccountRepository
	downstream *vault.VaultEntityDownstreamApi
	memStore   *io.MemoryStore
	txn        *io.MemoryStoreTxn
}

func NewObjectHandler(memStore *io.MemoryStore, txn *io.MemoryStoreTxn, api *vault.VaultEntityDownstreamApi) *ObjectHandler {
	return &ObjectHandler{
		eaRepo:     repo.NewEntityAliasRepo(txn),
		usersRepo:  iamrepos.NewUserRepository(txn),
		saRepo:     iamrepos.NewServiceAccountRepository(txn),
		memStore:   memStore,
		txn:        txn,
		downstream: api,
	}
}

func (h *ObjectHandler) HandleAuthSource(source *model.AuthSource) error {
	// todo crerate entity aliases for all users and all sources
	return nil
}

func (h *ObjectHandler) HandleEntity(entity *model.Entity) error {
	return h.process(entity)
}

func (h *ObjectHandler) HandleEntityAlias(entityAlias *model.EntityAlias) error {
	return h.process(entityAlias)
}

func (h *ObjectHandler) process(o io.MemoryStorableObject) error {
	// TODO Rewrite to normal handler
	actions, err := h.downstream.ProcessObject(h.memStore, h.txn, o)
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