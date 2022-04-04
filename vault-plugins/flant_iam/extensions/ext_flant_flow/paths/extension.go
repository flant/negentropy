package paths

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type flantFlowExtension struct {
	logger          hclog.Logger
	storage         *sharedio.MemoryStore
	liveConfig      *config.FlantFlowConfig
	liveConfigMutex sync.Mutex
}

func (e *flantFlowExtension) Logger() hclog.Logger {
	return e.logger
}

func FlantFlowDBSchema() *memdb.DBSchema {
	schema, err := repo.GetSchema()
	if err != nil {
		panic("error in flant_flow DBSchema:" + err.Error())
	}
	return schema
}

func FlantFlowPaths(ctx context.Context, conf *logical.BackendConfig, storage *sharedio.MemoryStore) ([]*framework.Path, error) {
	cfg, err := usecase.Config(storage.Txn(false)).GetConfig(ctx, conf.StorageView)
	if err != nil {
		return nil, err
	}

	b := &flantFlowExtension{
		logger:     conf.Logger.Named("FLOW"),
		storage:    storage,
		liveConfig: cfg,
	}

	paths := framework.PathAppend(
		teamPaths(b),
		teammatePaths(b),
		clientPaths(b),
		projectPaths(b),
		contactPaths(b),

		flantFlowConfigurePaths(b),
	)

	return paths, nil
}

func (e *flantFlowExtension) getLiveConfig() *config.FlantFlowConfig {
	e.liveConfigMutex.Lock()
	defer e.liveConfigMutex.Unlock()
	if e.liveConfig == nil {
		return nil
	}
	result := *e.liveConfig // make copy
	return &result
}

func (e *flantFlowExtension) setLiveConfig(cfg *config.FlantFlowConfig) {
	e.liveConfigMutex.Lock()
	defer e.liveConfigMutex.Unlock()
	if cfg == nil {
		// do nothing
		return
	}
	lc := *cfg // make copy
	e.liveConfig = &lc
}

// checkBaseConfigured run pathHandler if plugin is basically configured
func (e *flantFlowExtension) checkBaseConfigured(pathHandler framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		if err := e.getLiveConfig().IsBaseConfigured(); err != nil {
			return backentutils.ResponseErr(req, err)
		}
		return pathHandler(ctx, req, data)
	}
}

// checkConfigured run pathHandler if plugin is configured
func (e *flantFlowExtension) checkConfigured(pathHandler framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		if err := e.getLiveConfig().IsConfigured(); err != nil {
			return backentutils.ResponseErr(req, err)
		}
		return pathHandler(ctx, req, data)
	}
}

//
func (e *flantFlowExtension) checkFlantFlowClient(pathHandler framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		txn := e.storage.Txn(false)
		tenantRepo := iam_repo.NewTenantRepository(txn)
		tenantUUID := data.Get(clientUUIDKey).(string)
		tenant, err := tenantRepo.GetByID(tenantUUID)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if tenant.Origin != consts.OriginFlantFlow {
			return backentutils.ResponseErr(req, fmt.Errorf("%w:wrong client uuid", consts.ErrBadOrigin))
		}
		return pathHandler(ctx, req, data)
	}
}
