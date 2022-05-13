package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flant/negentropy/server-access/flant-server-accessd/config"
	"github.com/flant/negentropy/server-access/flant-server-accessd/db/sqlite"
	"github.com/flant/negentropy/server-access/flant-server-accessd/sync"
)

func main() {
	viper.SetDefault("author", "https://www.flant.com")

	rootCmd := &cobra.Command{
		Use:   "server_accessd",
		Short: "Flant negentropy  server-access daemon",
		Long: `Flant negentropy  server-access daemon
Find more information at https://flant.com`,
		Run: func(cmd *cobra.Command, args []string) {
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
		},
	}
	rootCmd.AddCommand(InitCMD())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
