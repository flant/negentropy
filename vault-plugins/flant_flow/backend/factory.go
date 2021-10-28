package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_clients"
	"github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	baseLogger := conf.Logger.ResetNamed("FLOW")

	logger := baseLogger.Named("Factory")
	logger.Debug("started")
	defer logger.Debug("exit")
	if os.Getenv("DEBUG") == "true" {
		logger.Debug("DEBUG mode is ON, messages will not be encrypted")
		sharedkafka.DoNotEncrypt = true
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	conf.Logger = baseLogger

	b, err := newBackend(conf)
	if err != nil {
		return nil, err
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	logger.Debug("normal finish")
	return b, nil
}

type ExtensionInitializationFunc func(ctx context.Context, initRequest *logical.InitializationRequest, storage *sharedio.MemoryStore) error

func initializer(storage *sharedio.MemoryStore) func(ctx context.Context, initRequest *logical.InitializationRequest) error {
	return func(ctx context.Context, initRequest *logical.InitializationRequest) error {
		initFuncs := []ExtensionInitializationFunc{}

		for _, f := range initFuncs {
			err := f(ctx, initRequest, storage)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func newBackend(conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	logger := conf.Logger.Named("newBackend")
	logger.Debug("started")
	defer logger.Debug("exit")

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView, conf.Logger)
	if err != nil {
		return nil, err
	}

	schema, err := repo.GetSchema()
	if err != nil {
		return nil, err
	}

	storage, err := sharedio.NewMemoryStore(schema, mb, conf.Logger)
	if err != nil {
		return nil, err
	}

	if backentutils.IsLoading(conf) {
		logger.Info("second run Factory, apply kafka operations on MemoryStore")

		// add here sources

		err = storage.Restore()
		if err != nil {
			return nil, err
		}

		// // add here destinations

		storage.RunKafkaSourceMainLoops()

	} else {
		logger.Info("first run Factory, skipping kafka operations on MemoryStore")
	}

	b.InitializeFunc = initializer(storage)

	// tokenController := sharedjwt.NewJwtController(
	//	storage,
	//	mb.GetEncryptionPublicKeyStrict,
	//	conf.Logger,
	//	time.Now,
	// )

	b.PeriodicFunc = func(ctx context.Context, request *logical.Request) error {
		periodicLogger := conf.Logger.Named("Periodic")
		periodicLogger.Debug("Run periodic function")
		defer periodicLogger.Debug("Periodic function finished")

		var allErrors *multierror.Error
		// run := func(controllerName string, action func() error) {
		//	periodicLogger.Debug(fmt.Sprintf("Run %s OnPeriodical", controllerName))
		//
		//	err := action()
		//	if err != nil {
		//		allErrors = multierror.Append(allErrors, err)
		//		periodicLogger.Error(fmt.Sprintf("Error while %s periodical function: %s", controllerName, err), "err", err)
		//		return
		//	}
		//
		//	periodicLogger.Debug(fmt.Sprintf("%s OnPeriodical was successful run", controllerName))
		// }

		// run("jwtController", func() error {
		//	tx := storage.Txn(true)
		//	defer tx.Abort()
		//
		//	err := tokenController.OnPeriodical(tx)
		//	if err != nil {
		//		return err
		//	}
		//
		//	if err := tx.Commit(); err != nil {
		//		periodicLogger.Error(fmt.Sprintf("Can not commit memdb transaction: %s", err), "err", err)
		//		return err
		//	}
		//
		//	return nil
		// })

		return allErrors
	}

	userclient, err := iam_clients.NewUserClient()
	if err != nil {
		return nil, err
	}

	tenantClient, err := iam_clients.NewTenantClient()
	if err != nil {
		return nil, err
	}

	b.Paths = framework.PathAppend(
		teamPaths(b, storage),
		teammatePaths(b, storage, fixtures.TeammateUUID1, userclient),
		clientPaths(b, storage, tenantClient),
		// tokenController.ApiPaths(),
	)

	b.Clean = func(context.Context) {
		l := b.Logger().Named("cleanup")
		l.Info("started")

		storage.Close()

		l.Info("finished")
	}

	logger.Debug("normal finish")
	return b, nil
}

const commonHelp = `
FLOW API here
`
