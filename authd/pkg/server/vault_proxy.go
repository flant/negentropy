package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/flant/negentropy/authd/pkg/config"
	utils "github.com/flant/negentropy/authd/pkg/util"
)

type VaultProxy struct {
	AuthdConfig       *config.AuthdConfig
	AuthdSocketConfig *config.AuthdSocketConfig

	SocketPath string
	Server     http.Server
	Router     chi.Router

	stopped bool
}

func NewVaultProxy(authdConfig *config.AuthdConfig, authdSocketConfig *config.AuthdSocketConfig) *VaultProxy {
	return &VaultProxy{
		AuthdConfig:       authdConfig,
		AuthdSocketConfig: authdSocketConfig,
		SocketPath:        createPath(authdSocketConfig.GetPath(), authdConfig.GetDefaultSocketDirectory()),
	}
}

func (v *VaultProxy) Start() error {
	sockCfg := v.AuthdSocketConfig
	address := v.SocketPath

	err := os.MkdirAll(path.Dir(address), os.FileMode(sockCfg.GetMode()))
	if err != nil {
		return fmt.Errorf("create directories for socket '%s': %v", address, err)
	}

	exists, err := utils.FileExists(address)
	if err != nil {
		return fmt.Errorf("check socket '%s': %v", address, err)
	}
	if exists {
		err = os.Remove(address)
		if err != nil {
			return fmt.Errorf("remove existing socket '%s': %v", address, err)
		}
	}

	// Create a socket listener.
	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("listen on '%s': %v", address, err)
	}

	logrus.Infof("Listen on %s.", address)

	v.Router = chi.NewRouter()
	v.Router.Use(NewStructuredLogger(address))
	v.Router.Use(DebugAwareLogger)
	v.Router.Use(middleware.Recoverer)

	SetupLoginHandler(v.Router, v.AuthdConfig, v.AuthdSocketConfig)

	v.Server = http.Server{
		Handler: v.Router,
	}

	go func() {
		if err := v.Server.Serve(listener); err != nil {
			if v.stopped {
				return
			}
			logrus.Errorf("Starting HTTP server for '%s': %v", address, err)
			os.Exit(1)
		}
	}()

	return nil
}

// Stop gracefully stops a server.
func (v *VaultProxy) Stop() {
	v.stopped = true
	idleConnsClosed := make(chan struct{})
	logrus.Debugf("Stop server on '%s'...", v.SocketPath)
	go func() {
		err := v.Server.Shutdown(context.Background())
		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.Warnf("Stop server on '%s': %v", v.SocketPath, err)
		}
		close(idleConnsClosed)
	}()

	<-idleConnsClosed
}

func createPath(path, defaultDir string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return filepath.Join(defaultDir, path)
}
