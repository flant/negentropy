package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/flant/negentropy/authd/pkg/daemon"
	"github.com/flant/negentropy/authd/pkg/log"
	"github.com/flant/negentropy/authd/pkg/signal"
)

func main() {
	kpApp := kingpin.New("authd", "")

	startCmd := kpApp.Command("start", "Start authd.").
		Default().
		Action(func(c *kingpin.ParseContext) error {
			logrus.SetOutput(os.Stdout)
			logrus.SetFormatter(&logrus.TextFormatter{
				DisableColors: true,
				FullTimestamp: true,
			})
			log.DebugLogger().SetFormatter(&logrus.TextFormatter{
				DisableColors:    true,
				FullTimestamp:    true,
				CallerPrettyfier: log.DebugCallerPrettyfier,
			})

			authd := daemon.NewDefaultAuthd()
			err := authd.Start()
			if err != nil {
				fmt.Printf("Start: %v\n", err)
				os.Exit(1)
			}

			// listen for SIGTERM and block
			signal.WaitForProcessInterruption(func() {
				authd.Stop()
				os.Exit(1)
			})
			return nil
		})
	daemon.DefineFlags(startCmd, daemon.DefaultConfig)

	kingpin.MustParse(kpApp.Parse(os.Args[1:]))
}
