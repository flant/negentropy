package daemon

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/flant/negentropy/authd/pkg/config"
	"github.com/flant/negentropy/authd/pkg/jwt"
	"github.com/flant/negentropy/authd/pkg/server"
	"github.com/flant/negentropy/authd/pkg/util"
	"github.com/flant/negentropy/authd/pkg/vault"
)

type Config struct {
	ConfDirectory string
}

type Authd struct {
	Config             *Config
	AuthdConfig        *config.AuthdConfig
	AuthdSocketConfigs []*config.AuthdSocketConfig

	Servers []*server.VaultProxy

	stop      chan struct{}
	refresher *util.PostponedRetryLoop

	refreshLoopCtx    context.Context
	refreshLoopCancel context.CancelFunc
	refreshLoopDone   chan struct{}
}

// Start runs servers for all defined socket configurations.
func (a *Authd) Start() error {
	// Load configs from Config.ConfDirectory
	confFiles, err := config.RecursiveFindConfFiles(a.Config.ConfDirectory)
	if err != nil {
		return err
	}

	a.AuthdConfig, a.AuthdSocketConfigs, err = config.LoadConfigFiles(confFiles)
	if err != nil {
		return err
	}

	if len(a.AuthdSocketConfigs) == 0 {
		return fmt.Errorf("no socket configurations loaded from %s", a.Config.ConfDirectory)
	}

	// Load and check JWT.
	err = jwt.DefaultStorage.Load(a.AuthdConfig.GetJWTPath())
	if err != nil {
		if errors.Is(err, jwt.ExpiredErr) {
			return fmt.Errorf("JWT at '%s' is expired. Update manually.", a.AuthdConfig.GetJWTPath())
		}
		return err
	}

	a.refreshLoopCtx, a.refreshLoopCancel = context.WithCancel(context.Background())
	a.refreshLoopDone = make(chan struct{})

	go a.RunRefreshLoop(a.refreshLoopCtx, a.AuthdConfig)

	// Start one server for one AuthdSocketConfig
	a.Servers = make([]*server.VaultProxy, 0)
	for _, socketConfig := range a.AuthdSocketConfigs {
		srv := server.NewVaultProxy(a.AuthdConfig, socketConfig)
		err := srv.Start()
		if err != nil {
			return err
		}
		a.Servers = append(a.Servers, srv)
	}

	return nil
}

// Stop
func (a *Authd) Stop() {
	if a.refreshLoopCancel != nil {
		a.refreshLoopCancel()
	}
	for _, srv := range a.Servers {
		srv.Stop()
	}
	<-a.refreshLoopDone
}

// RunRefreshLoop
func (a *Authd) RunRefreshLoop(ctx context.Context, authdConfig *config.AuthdConfig) {
	authServerAddr, err := config.GetDefaultServerAddr(authdConfig.GetServers(), "auth")
	if err != nil {
		logrus.Errorf("JWT refresh not started: no 'auth' server in configuration: %v", err)
		return
	}

	defer close(a.refreshLoopDone)
	for {
		logrus.Debugf("RunRefreshLoop: CreateRefresher")
		refresher := jwt.DefaultStorage.CreateRefresher(func(ctx context.Context) error {
			vaultCl := vault.NewClient(authServerAddr)
			// It is time to refresh. Check if current JWT is not expired.
			currJWT, err := jwt.DefaultStorage.GetJWT()
			if err != nil {
				logrus.Errorf("Refresh JWT: token is expired, update manually!")
				return util.StopRetriesErr
			}

			logrus.Debugf("Try to obtain new JWT...")

			newJWT, err := vaultCl.RefreshJWT(ctx, currJWT)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logrus.Errorf("Refresh JWT: %v", err)
				}
				return err
			}
			logrus.Debugf("New JWT is obtained.")

			err = jwt.DefaultStorage.Update(newJWT)
			if err != nil {
				logrus.Errorf("Update JWT: %v", err)
				return err
			}
			return nil
		})
		// There are 3 possibilities:
		// - context is cancelled -> just return to close a channel.
		// - token refreshed -> next iteration
		// - token becomes expired, so refresh is not possible anymore -> wait until user updates JWT.
		logrus.Debugf("RunRefreshLoop: RunLoop. First refresh at: %s", refresher.StartAfter.Format(time.RFC3339))
		err := refresher.RunLoop(ctx)
		if err != nil && errors.Is(err, context.Canceled) {
			return
		}
		// Check if token is expired. Wait until token becomes active or ctx is canceled.
		err = waitForActiveJWT(ctx)
		if err != nil && errors.Is(err, context.Canceled) {
			return
		}
	}
}

func waitForActiveJWT(ctx context.Context) error {
	_, err := jwt.DefaultStorage.GetJWT()
	logrus.Debugf("waitForActiveJWT: first check if JWT is expired err: %v", err)
	if err == nil {
		return nil
	}

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			_, err := jwt.DefaultStorage.GetJWT()
			logrus.Debugf("waitForActiveJWT: check if JWT is expired err: %v", err)
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			logrus.Debug("waitForActiveJWT: ctx cancel")
			return ctx.Err()
		}
	}
}
