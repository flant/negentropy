package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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
	//JwtAccessor JwtAccessor // ?
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
		fmt.Printf("Debug HTTP server fail to create socket '%s': %v", address, err)
		return err
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

	fmt.Printf("Listen on %s\n", address)

	v.Router = chi.NewRouter()
	v.Router.Use(NewStructuredLogger(address))
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
			fmt.Printf("Error starting Debug HTTP server: %s", err)
			os.Exit(1)
		}
	}()

	return nil
}

// Stop gracefully stops a server.
func (v *VaultProxy) Stop() {
	v.stopped = true
	idleConnsClosed := make(chan struct{})
	fmt.Printf("Stop server on '%s'\n", v.SocketPath)
	go func() {
		if err := v.Server.Shutdown(context.Background()); err != nil {
			fmt.Printf("WARN: stop server on '%s': %v\n", v.SocketPath, err)
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
