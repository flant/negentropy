package backend

import (
	"context"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const PluginName = "flant_servers"

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	b, err := newBackend(conf)
	if err != nil {
		return nil, err
	}
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend(conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		BackendType: logical.TypeLogical,
	}

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView, "root_source")
	if err != nil {
		return nil, err
	}

	schema, err := model.GetSchema()
	if err != nil {
		return nil, err
	}

	for name, table := range model.ServerSchema().Tables {
		if _, ok := schema.Tables[name]; ok {
			return nil, fmt.Errorf("table %q already there", name)
		}
		schema.Tables[name] = table
	}

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		return nil, err
	}

	storage.SetLogger(conf.Logger)
	storage.AddKafkaSource(kafka_source.NewMainKafkaSource(mb, "root_source"))

	if _, ok := conf.Config["testRun"]; ok {
		err := fillDBWithTenantsAndProjects(storage, conf.Config["testTenantUUID"], conf.Config["testProjectUUID"])
		if err != nil {
			return nil, err
		}
	}

	// TODO kafka sync
	/*
		err = storage.Restore()
			if err != nil {
				return nil, err
			}
	*/

	b.Paths = framework.PathAppend(
		backend.serverPaths(b, storage))

	return b, nil
}

func fillDBWithTenantsAndProjects(storage *sharedio.MemoryStore, tenantUUID, projectUUID string) error {
	tx := storage.Txn(true)
	defer tx.Abort()

	err := tx.Insert(model.TenantType, &model.Tenant{
		UUID:       tenantUUID,
		Version:    model.NewResourceVersion(),
		Identifier: "test",
	})
	if err != nil {
		return err
	}

	err = tx.Insert(model.ProjectType, &model.Project{
		UUID:       projectUUID,
		TenantUUID: tenantUUID,
		Version:    model.NewResourceVersion(),
		Identifier: "test",
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}
