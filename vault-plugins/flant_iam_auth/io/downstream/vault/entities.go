package vault

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
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
	loggerFactory func() log.Logger
}

func NewVaultEntityDownstreamApi(getClient io.BackoffClientGetter, mountAccessorGetter *MountAccessorGetter, loggerFactory func() log.Logger) *VaultEntityDownstreamApi {
	return &VaultEntityDownstreamApi{
		getClient:           getClient,
		mountAccessorGetter: mountAccessorGetter,
		loggerFactory: loggerFactory,
	}
}

func (a *VaultEntityDownstreamApi) ProcessObject(ms *io.MemoryStore, txn *io.MemoryStoreTxn, obj io.MemoryStorableObject) ([]io.DownstreamAPIAction, error) {
	switch obj.ObjType() {
	case model.EntityType:
		entity, ok := obj.(*model.Entity)
		if !ok {
			return nil, fmt.Errorf("does not cast to Entity")
		}
		return a.ProcessEntity(ms, txn, entity)
	case model.EntityAliasType:
		entityAlias, ok := obj.(*model.EntityAlias)
		if !ok {
			return nil, fmt.Errorf("does not cast to EntityAlias")
		}
		return a.ProcessEntityAlias(ms, txn, entityAlias)
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
		return a.ProcessDeleteEntity(ms, txn, entity.Name)
	case model.EntityAliasType:
		entityAlias, ok := obj.(*model.EntityAlias)
		if !ok {
			return nil, fmt.Errorf("does not cast to EntityAlias")
		}
		return a.ProcessDeleteEntityAlias(ms, txn, entityAlias.Name)
	}

	return make([]io.DownstreamAPIAction, 0), nil
}

func (a *VaultEntityDownstreamApi) ProcessEntity(ms *io.MemoryStore, txn *io.MemoryStoreTxn, entity *model.Entity) ([]io.DownstreamAPIAction, error) {
	clientApi, err := a.getClient()
	if err != nil {
		return nil, err
	}

	a.loggerFactory().Debug("Creating entity with name %s", entity.Name)
	action := io.NewVaultApiAction(func() error {
		return api.NewIdentityAPIWithBackOff(clientApi, backOffSettings).EntityApi().Create(entity.Name)
	})

	return []io.DownstreamAPIAction{action}, nil
}

func (a *VaultEntityDownstreamApi) ProcessEntityAlias(ms *io.MemoryStore, txn *io.MemoryStoreTxn, entityAlias *model.EntityAlias) ([]io.DownstreamAPIAction, error) {
	// got current snapshot in mem db
	readTxn := ms.Txn(false)

	// next we need to get vault entity id
	// we dont store them in memstorage, because this id
	// getting after creating entity in vault and store entity id
	// does not atomic

	// first, get entity for user id
	entityRepo := model.NewEntityRepo(readTxn)
	entity, err := entityRepo.GetByUserId(entityAlias.UserId)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("not found entity %v", entityAlias.UserId)
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
		return nil, fmt.Errorf("not found entity id for %s", entity.Name)
	}

	// getting mount accessor - identifer for mount point plugin
	mountAccessor, err := a.mountAccessorGetter.MountAccessor()
	if err != nil {
		return nil, err
	}

	a.loggerFactory().Debug("Creating entity with name %s for entity id % with mount accessor %s", entityAlias.Name, entityId, mountAccessor)
	action := io.NewVaultApiAction(func() error {
		return identityApi.AliasApi().Create(entityAlias.Name, entityId, mountAccessor)
	})

	return []io.DownstreamAPIAction{action}, nil
}

func (a *VaultEntityDownstreamApi) ProcessDeleteEntity(ms *io.MemoryStore, txn *io.MemoryStoreTxn, entityName string) ([]io.DownstreamAPIAction, error) {
	apiClient, err := a.getClient()
	if err != nil {
		return nil, err
	}

	a.loggerFactory().Debug("Delete entity with name %s", entityName)
	action := io.NewVaultApiAction(func() error {
		return api.NewIdentityAPIWithBackOff(apiClient, backOffSettings).EntityApi().DeleteByName(entityName)
	})

	return []io.DownstreamAPIAction{action}, nil
}

func (a *VaultEntityDownstreamApi) ProcessDeleteEntityAlias(ms *io.MemoryStore, txn *io.MemoryStoreTxn, entityAliasName string) ([]io.DownstreamAPIAction, error) {
	apiClient, err := a.getClient()
	if err != nil {
		return nil, err
	}

	// getting mount accessor - identifer for mount point plugin
	mountAccessor, err := a.mountAccessorGetter.MountAccessor()
	if err != nil {
		return nil, err
	}

	a.loggerFactory().Debug("Delete entity alias with name %s", entityAliasName)
	action := io.NewVaultApiAction(func() error {
		return api.NewIdentityAPIWithBackOff(apiClient, backOffSettings).AliasApi().DeleteByName(entityAliasName, mountAccessor)
	})

	return []io.DownstreamAPIAction{action}, nil
}
