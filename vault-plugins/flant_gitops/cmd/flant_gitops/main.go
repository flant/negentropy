package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"

	"github.com/flant/negentropy/vault-plugins/flant_gitops"
)

func main() {
	logFileName := "flant_gitops.log"
	if v := os.Getenv("FLANT_GITOPS_LOG_FILE"); v != "" {
		logFileName = v
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("failed to open trdl.log file: %s", err)) // nolint:panic_check
	}

	hclog.DefaultOptions = &hclog.LoggerOptions{
		Level:           hclog.Trace,
		IncludeLocation: true,
		Output:          logFile,
	}

	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	if err := flags.Parse(os.Args[1:]); err != nil {
		hclog.L().Error("bad arguments", "error", err)
		os.Exit(1)
	}

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	err = plugin.Serve(&plugin.ServeOpts{
		Logger:             hclog.Default(),
		BackendFactoryFunc: flant_gitops.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	})
	if err != nil {
		hclog.L().Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
