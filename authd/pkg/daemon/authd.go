package daemon

import (
	"fmt"
	"github.com/flant/negentropy/authd/pkg/jwt"

	"github.com/flant/negentropy/authd/pkg/config"
	"github.com/flant/negentropy/authd/pkg/server"
)

type Config struct {
	ConfDirectory string
}

type Authd struct {
	Config             *Config
	AuthdConfig        *config.AuthdConfig
	AuthdSocketConfigs []*config.AuthdSocketConfig

	Servers []*server.VaultProxy
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
		return err
	}

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

	fmt.Printf("autht started with conf dir: '%s'\n", a.Config.ConfDirectory)
	return nil
}

// Stop
func (a *Authd) Stop() {
	for _, srv := range a.Servers {
		srv.Stop()
	}
}
