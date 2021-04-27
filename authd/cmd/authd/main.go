package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/flant/negentropy/authd/pkg/daemon"
	"github.com/flant/negentropy/authd/pkg/signal"
)

func main() {
	kpApp := kingpin.New("authd", "")

	startCmd := kpApp.Command("start", "Start authd.").
		Default().
		Action(func(c *kingpin.ParseContext) error {
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
