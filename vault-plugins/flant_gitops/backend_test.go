package flant_gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/kube"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/task_manager"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

type TestableBackend struct {
	B               *backend
	Storage         logical.Storage
	Logger          *util.TestLogger
	Clock           *util.MockClock
	MockKubeService *kube.MockKubeService
}

// getTestBackend prepare and returns test backend with mocked systemClock
func getTestBackend(ctx context.Context) (*TestableBackend, error) {
	mockedSystemClock, systemClockMock := util.NewMockedClock(time.Now())
	systemClock = mockedSystemClock // replace value of global variable for system time operating

	var kubeServiceMock *kube.MockKubeService
	kubeServiceProvider, kubeServiceMock = kube.NewMock()

	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	logical.TestBackendConfig()

	testLogger := util.NewTestLogger()

	storage := &logical.InmemStorage{}
	config := &logical.BackendConfig{
		Logger: testLogger.VaultLogger,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: storage,
	}

	b, err := newBackend(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create backend: %w", err)
	}

	if err := b.SetupBackend(ctx, config); err != nil {
		return nil, fmt.Errorf("unable to setup backend: %s", err)
	}

	taskManagerServiceProvider = task_manager.NewMock(b.Backend)

	return &TestableBackend{
		B:               b,
		Storage:         storage,
		Logger:          testLogger,
		Clock:           systemClockMock,
		MockKubeService: kubeServiceMock,
	}, nil
}
