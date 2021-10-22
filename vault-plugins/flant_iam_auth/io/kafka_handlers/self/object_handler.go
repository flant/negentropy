package self

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"

	iamrepos "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type ObjectHandler struct {
	vaultEntityDownstream *vault.VaultEntityDownstreamApi
	logger                hclog.Logger
}

func NewObjectHandler(api *vault.VaultEntityDownstreamApi, parentLogger hclog.Logger) *ObjectHandler {
	return &ObjectHandler{
		vaultEntityDownstream: api,
		logger:                parentLogger.Named("SelfSourceHandler"),
	}
}

func (h *ObjectHandler) HandleAuthSource(txn *io.MemoryStoreTxn, source *model.AuthSource) error {
	l := h.logger
	l.Debug("Handle auth source", source.Name)
	usersRepo := iam_repo.NewUserRepository(txn)
	eaRepo := repo.NewEntityAliasRepo(txn)
	saRepo := iam_repo.NewServiceAccountRepository(txn)

	err := usersRepo.Iter(func(user *iamrepos.User) (bool, error) {
		l.Debug(fmt.Sprintf("Create new ea mem object for user %s and source %s", user.FullIdentifier, source.Name))
		err := eaRepo.CreateForUser(user, source)
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

	return saRepo.Iter(func(account *iamrepos.ServiceAccount) (bool, error) {
		l.Debug("Create new ea mem object for SA and source", account.FullIdentifier, source.Name)
		err := eaRepo.CreateForSA(account, source)
		if err != nil {
			l.Error("Cannot create ea mem object for SA an source", account.FullIdentifier, source.Name, err)
			return false, nil
		}

		l.Debug("Created new ea mem object for SA and source", account.FullIdentifier, source.Name)
		return true, nil
	})
}

func (h *ObjectHandler) HandleEntity(txn *io.MemoryStoreTxn, entity *model.Entity) error {
	return h.processActions(h.vaultEntityDownstream.ProcessEntity(txn, entity))
}

func (h *ObjectHandler) HandleEntityAlias(txn *io.MemoryStoreTxn, entityAlias *model.EntityAlias) error {
	return h.processActions(h.vaultEntityDownstream.ProcessEntityAlias(txn, entityAlias))
}

func (h *ObjectHandler) DeletedAuthSource(txn *io.MemoryStoreTxn, uuid string) error {
	l := h.logger
	l.Debug("Handle delete source", uuid)
	eaRepo := repo.NewEntityAliasRepo(txn)

	err := eaRepo.GetBySource(uuid, func(alias *model.EntityAlias) (bool, error) {
		l.Debug("Delete entity alias obj", alias.UUID, alias.Name)
		err := eaRepo.DeleteByID(alias.UUID)
		if err != nil {
			l.Debug("Can not delete entity alias obj", alias.UUID, alias.Name, err)
			return false, err
		}
		l.Debug("Deleted entity alias obj", alias.UUID, alias.Name)
		return true, nil
	})

	return err
}

func (h *ObjectHandler) DeletedEntity(txn *io.MemoryStoreTxn, id string) error {
	return h.processActions(h.vaultEntityDownstream.ProcessDeleteEntity(txn, id))
}

func (h *ObjectHandler) DeletedEntityAlias(txn *io.MemoryStoreTxn, id string) error {
	actions, err := h.vaultEntityDownstream.ProcessDeleteEntityAlias(txn, id)
	return h.processActions(actions, err)
}

func (h *ObjectHandler) processActions(actions []io.DownstreamAPIAction, err error) error {
	if err != nil {
		h.logger.Error(fmt.Sprintf("Receive error from action processor: %v", err), "err", err)
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

type ModelHandler interface {
	HandleAuthSource(txn *io.MemoryStoreTxn, user *model.AuthSource) error
	HandleEntity(txn *io.MemoryStoreTxn, entity *model.Entity) error
	HandleEntityAlias(txn *io.MemoryStoreTxn, entity *model.EntityAlias) error

	DeletedAuthSource(txn *io.MemoryStoreTxn, uuid string) error
	DeletedEntity(txn *io.MemoryStoreTxn, uuid string) error
	DeletedEntityAlias(txn *io.MemoryStoreTxn, uuid string) error
}

func HandleNewMessageSelfSource(txn *io.MemoryStoreTxn, handler ModelHandler, msg *sharedkafka.MsgDecoded) error {
	isDelete := msg.IsDeleted()

	var inputObject interface{}
	var entityHandler func() error

	objId := msg.ID

	switch msg.Type {
	case model.AuthSourceType:
		source := &model.AuthSource{}
		inputObject = source
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedAuthSource(txn, objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleAuthSource(txn, source)
			}
		}

	case model.EntityType:
		entity := &model.Entity{}
		inputObject = entity
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedEntity(txn, objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleEntity(txn, entity)
			}
		}

	case model.EntityAliasType:
		entityAlias := &model.EntityAlias{}
		inputObject = entityAlias
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedEntityAlias(txn, objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleEntityAlias(txn, entityAlias)
			}
		}

	case model.AuthMethodType, model.MethodTypeJWT:
		// don't need handle
		return nil

	default:
		return nil
	}

	if !isDelete {
		// only unmarshal this object in mem storage
		// because we read here from self storage
		err := json.Unmarshal(msg.Data, inputObject)
		if err != nil {
			return err
		}
	}

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}
