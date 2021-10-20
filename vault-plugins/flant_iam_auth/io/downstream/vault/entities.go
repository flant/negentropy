package vault

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const maxElapsedTime = 5 * time.Second

func backOffSettings() backoff.BackOff {
	backoffRequest := backoff.NewExponentialBackOff()
	backoffRequest.MaxElapsedTime = maxElapsedTime
	return backoffRequest
}

type VaultEntityDownstreamApi struct {
	getClient           io.BackoffClientGetter
	mountAccessorGetter *MountAccessorGetter
	logger              log.Logger
}

func NewVaultEntityDownstreamApi(getClient io.BackoffClientGetter, mountAccessorGetter *MountAccessorGetter, parenatLogger log.Logger) *VaultEntityDownstreamApi {
	return &VaultEntityDownstreamApi{
		getClient:           getClient,
		mountAccessorGetter: mountAccessorGetter,
		logger:              parenatLogger.Named("VaultIdentityClient"),
	}
}

func (a *VaultEntityDownstreamApi) ProcessObject(txn *io.MemoryStoreTxn, obj io.MemoryStorableObject) ([]io.DownstreamAPIAction, error) {
	switch obj.ObjType() {
	case model.EntityType:
		entity, ok := obj.(*model.Entity)
		if !ok {
			return nil, fmt.Errorf("does not cast to Entity")
		}
		return a.ProcessEntity(txn, entity)
	case model.EntityAliasType:
		entityAlias, ok := obj.(*model.EntityAlias)
		if !ok {
			return nil, fmt.Errorf("does not cast to EntityAlias")
		}
		return a.ProcessEntityAlias(txn, entityAlias)
	}

	return make([]io.DownstreamAPIAction, 0), nil
}

func (a *VaultEntityDownstreamApi) ProcessObjectDelete(ms *io.MemoryStore, txn *io.MemoryStoreTxn, obj io.MemoryStorableObject) ([]io.DownstreamAPIAction, error) {
	switch obj.ObjType() {
	case model.EntityType:
		entity, ok := obj.(*model.Entity)
		if !ok {
			return nil, fmt.Errorf("does not cast to Entity")
		}
		return a.ProcessDeleteEntity(txn, entity.Name)
	case model.EntityAliasType:
		entityAlias, ok := obj.(*model.EntityAlias)
		if !ok {
			return nil, fmt.Errorf("does not cast to EntityAlias")
		}
		return a.ProcessDeleteEntityAlias(txn, entityAlias.Name)
	}

	return make([]io.DownstreamAPIAction, 0), nil
}

func (a *VaultEntityDownstreamApi) ProcessEntity(txn *io.MemoryStoreTxn, entity *model.Entity) ([]io.DownstreamAPIAction, error) {
	err := a.createEntityInMemoryStoreIfNotExists(txn, entity)
	if err != nil {
		return nil, err
	}
	clientApi, err := a.getClient()
	if err != nil {
		return nil, err
	}

	action := io.NewVaultApiAction(func() error {
		a.logger.Debug(fmt.Sprintf("Creating vault entity with name %s", entity.Name), "name", entity.Name)
		err := api.NewIdentityAPIWithBackOff(clientApi, backOffSettings).EntityApi().Create(entity.Name)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Cannot create vault entity with name %s: %v", entity.Name, err), "name", entity.Name, "err", err)
			return err
		}

		a.logger.Debug(fmt.Sprintf("Created vault entity with name %s", entity.Name), "name", entity.Name)
		return nil
	})

	return []io.DownstreamAPIAction{action}, nil
}

// this func needs to proceed vault downing case when incoming from kafka-root-source user was proceeded,
// but incoming from kafka-self-source entity && entity-alias were not
func (a *VaultEntityDownstreamApi) createEntityInMemoryStoreIfNotExists(txn *io.MemoryStoreTxn, entity *model.Entity) error {
	entityRepo := repo.NewEntityRepo(txn)
	// e, err := entityRepo.GetByUserId(entity.UserId)
	e, err := entityRepo.GetByID(entity.UUID)
	if err != nil {
		return fmt.Errorf("getting entity from MemoryStore:%w", err)
	}
	if e == nil {
		a.logger.Warn(fmt.Sprintf("fixing absence entity: %#v", *entity))
		err = entityRepo.Put(entity)
		if err != nil {
			return fmt.Errorf("creating entity in MemoryStore:%w", err)
		}
		a.logger.Debug(fmt.Sprintf("entity UUID=%s created", entity.UUID))
	}
	return nil
}

func (a *VaultEntityDownstreamApi) ProcessEntityAlias(txn *io.MemoryStoreTxn, entityAlias *model.EntityAlias) ([]io.DownstreamAPIAction, error) {
	// next we need to get vault entity id
	// we dont store them in memstorage, because this id
	// getting after creating entity in vault and store entity id
	// does not atomic

	// first, get entity for user id
	entityRepo := repo.NewEntityRepo(txn)
	entity, err := entityRepo.GetByUserId(entityAlias.UserId)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		a.logger.Error(fmt.Sprintf("Cannot get entity entity alias %s: %v", entityAlias.Name, err), "name", entityAlias.Name, "err", err)
		return nil, fmt.Errorf("not found entity %v", entityAlias.UserId)
	}

	err = a.createEntityAliasInMemoryStoreIfNotExists(txn, entityAlias)
	if err != nil {
		return nil, err
	}

	apiClient, err := a.getClient()
	if err != nil {
		return nil, err
	}

	// getting entity id through vault api (with backoff)
	identityApi := api.NewIdentityAPIWithBackOff(apiClient, backOffSettings)

	entityId, err := identityApi.EntityApi().GetID(entity.Name)
	if err != nil {
		return nil, err
	}

	if entityId == "" {
		a.logger.Error(fmt.Sprintf("Cannot get vault entity id for entity %s: %v", entity.Name, err), "name", entity.Name, "err", err)
		return nil, fmt.Errorf("not found entity id for %s", entity.Name)
	}

	// getting mount accessor - identifer for mount point plugin
	mountAccessor, err := a.mountAccessorGetter.MountAccessor()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Cannot get mount accessor: %v", err), "name", entityAlias.Name, "err", err)
		return nil, err
	}

	action := io.NewVaultApiAction(func() error {
		a.logger.Debug(
			fmt.Sprintf("Creating entity alias with name %s for entity id %s with mount accessor %s", entityAlias.Name, entityId, mountAccessor),
			"eaName", entityAlias.Name, "entityId", entityId, "ma", mountAccessor,
		)

		err := identityApi.AliasApi().Create(entityAlias.Name, entityId, mountAccessor)
		if err != nil {
			a.logger.Error(
				fmt.Sprintf(
					"Can not create entity alias with name %s for entity id %s with mount accessor %s: %v",
					entityAlias.Name, entityId, mountAccessor, err,
				),
				"eaName", entityAlias.Name, "entityId", entityId, "ma", mountAccessor, "err", err,
			)

			return err
		}

		a.logger.Debug(
			fmt.Sprintf("Entity alias %s created for entity id %s with mount accessor %s", entityAlias.Name, entityId, mountAccessor),
			"eaName", entityAlias.Name, "entityId", entityId, "ma", mountAccessor,
		)

		return nil
	})

	return []io.DownstreamAPIAction{action}, nil
}

// this func needs to proceed vault downing case when incoming from kafka-root-source user was proceeded,
// but incoming from kafka-self-source  entity-alias was not
func (a *VaultEntityDownstreamApi) createEntityAliasInMemoryStoreIfNotExists(txn *io.MemoryStoreTxn, entityAlias *model.EntityAlias) error {
	eaRepo := repo.NewEntityAliasRepo(txn)
	ea, err := eaRepo.GetById(entityAlias.UUID)
	if err != nil {
		return fmt.Errorf("getting entityAlias from MemoryStore:%w", err)
	}
	if ea == nil {
		a.logger.Warn(fmt.Sprintf("fixing absence entityAlias: %#v", *entityAlias))
		err = eaRepo.Put(entityAlias)
		if err != nil {
			return fmt.Errorf("creating entityAlias in MemoryStore:%w", err)
		}
		a.logger.Debug(fmt.Sprintf("entity UUID=%s created", entityAlias.UUID))

	}
	return nil
}

func (a *VaultEntityDownstreamApi) ProcessDeleteEntity(txn *io.MemoryStoreTxn, entityName string) ([]io.DownstreamAPIAction, error) {
	apiClient, err := a.getClient()
	if err != nil {
		return nil, err
	}

	action := io.NewVaultApiAction(func() error {
		a.logger.Debug(fmt.Sprintf("Deleting entity with name %s", entityName), "entityName", entityName)
		err := api.NewIdentityAPIWithBackOff(apiClient, backOffSettings).EntityApi().DeleteByName(entityName)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Can not delete entity %s: %v", entityName, err), "entityName", entityName, "err", err)
			return err
		}

		a.logger.Debug(fmt.Sprintf("Entity %s deleted", entityName), "entityName", entityName)

		return nil
	})

	return []io.DownstreamAPIAction{action}, nil
}

func (a *VaultEntityDownstreamApi) ProcessDeleteEntityAlias(txn *io.MemoryStoreTxn, entityAliasName string) ([]io.DownstreamAPIAction, error) {
	apiClient, err := a.getClient()
	if err != nil {
		return nil, err
	}

	// getting mount accessor - identifer for mount point plugin
	mountAccessor, err := a.mountAccessorGetter.MountAccessor()
	if err != nil {
		return nil, err
	}

	action := io.NewVaultApiAction(func() error {
		a.logger.Debug(fmt.Sprintf("Deleting entity alias a with name %s", entityAliasName), "eaName", entityAliasName)
		err := api.NewIdentityAPIWithBackOff(apiClient, backOffSettings).AliasApi().DeleteByName(entityAliasName, mountAccessor)
		if err != nil {
			a.logger.Error(fmt.Sprintf("Can not delete entity alias %s: %v", entityAliasName, err), "eaName", entityAliasName, "err", err)
			return err
		}
		a.logger.Debug(fmt.Sprintf("Entity alias %s deleted", entityAliasName), "eaName", entityAliasName)
		return nil
	})

	return []io.DownstreamAPIAction{action}, nil
}
