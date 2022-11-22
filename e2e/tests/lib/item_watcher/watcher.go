// this app:
//  1) watch over kafka topics and collect information over message headers
//  2) clean topic by run /clean/{topic}
package main

import (
	"context"
	"flag"
	"github.com/flant/negentropy/e2e/tests/lib"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/flant/negentropy/e2e/tests/lib/item_watcher/internal"
)

func main() {
	log.SetFlags(0)

	var listenAddress = flag.String("listen", ":3333", "Listen address.")

	flag.Parse()

	if flag.NArg() != 0 {
		flag.Usage()
		log.Fatalf("\nERROR You MUST NOT pass any positional arguments")
	}

	server := internal.WatcherServer{
		Edge: time.Now(),
		Topics: []internal.Topic{
			{
				Name: "auth_source.auth",
				Type: internal.AuthPluginSelfTopic,
				OriginVault: &internal.Vault{
					Url:       lib.GetAuthVaultUrl(),
					RootToken: lib.GetAuthRootToken(),
				},
			},
			{
				Name: "auth_source.root",
				Type: internal.AuthPluginSelfTopic,
				OriginVault: &internal.Vault{
					Url:       lib.GetRootVaultUrl(),
					RootToken: lib.GetRootRootToken(),
				},
			},
			{
				Name:        "jwks",
				Type:        internal.JwksTopic,
				OriginVault: nil, // clean is not applicable
			},
			{
				Name:        "multipass_generation_num",
				Type:        internal.MultipassNumTopic,
				OriginVault: nil, // clean is not applicable
			},
			{
				Name: "root_source",
				Type: internal.IamPluginSelfTopic,
				OriginVault: &internal.Vault{
					Url:       lib.GetRootVaultUrl(),
					RootToken: lib.GetRootRootToken(),
				},
			},
			{
				Name:        "root_source.auth",
				Type:        internal.AuthPluginRootReplicaTopic,
				OriginVault: nil, // clean is not applicable
			},
			{
				Name:        "root_source.root",
				Type:        internal.AuthPluginRootReplicaTopic,
				OriginVault: nil, // clean is not applicable
			},
		},
		ListenAddress: *listenAddress,
	}
	err := server.InitServer()
	if err != nil {
		log.Fatalf("init: %s", err.Error())
	}

	go func() {
		server.RunServer()
	}()
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		log.Println("os.Interrupt")
	case <-server.ShutDownRequest:
		log.Println("shutdown request come")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
