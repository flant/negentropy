package main

import (
	"context"
	"flag"
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
		Topics: []string{
			"auth_source.auth",
			"auth_source.root",
			"jwks",
			"multipass_generation_num",
			"root_source",
			"root_source.auth",
			"root_source.root",
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

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
