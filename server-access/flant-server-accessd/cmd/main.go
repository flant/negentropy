package main

import (
	"log"
	"os"

	"github.com/flant/server-access/flant-server-accessd/config"
	"github.com/flant/server-access/flant-server-accessd/db/sqlite"
	"github.com/flant/server-access/flant-server-accessd/sync"
)

func main() {
	var err error

	config.AppConfig, err = config.LoadConfig(os.Getenv("SERVER_ACCESSD_CONF"))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	database, err := sqlite.NewUserDatabase(config.AppConfig.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}

	err = database.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	syncer := sync.NewPeriodic(database, config.AppConfig.AuthdSettings, config.AppConfig.ServerAccessSettings)
	syncer.Start()

	lockCh := make(chan struct{}, 0)
	<-lockCh

	//syncServer := sync.NewServer(":8080", "/v1/sync", database)
	//err = syncServer.Start()
	//if err != nil {
	//	log.Fatal(err)
	//}
}
