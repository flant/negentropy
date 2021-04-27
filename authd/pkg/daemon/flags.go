package daemon

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

func DefineFlags(cmd *kingpin.CmdClause, config *Config) {
	cmd.Flag("conf-dir", "A path to directory with configuration files.").
		Envar("AUTHD_CONF_DIRECTORY").
		Default(config.ConfDirectory).
		StringVar(&config.ConfDirectory)
}
