package paths

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type flantFlowExtension struct {
	logger hclog.Logger
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

func FlantFlowPaths(conf *logical.BackendConfig, storage *sharedio.MemoryStore) []*framework.Path {
	b := &flantFlowExtension{
		logger: conf.Logger.Named("FLOW"),
	}

	paths := framework.PathAppend(
		teamPaths(b, storage),
		teammatePaths(b, storage, fixtures.TeammateUUID1, nil),
		clientPaths(b, storage, nil),
		projectPaths(b, storage, nil),
		contactPaths(b, storage, nil),
	)

	return paths
}
