package config

import (
	"bytes"
	"fmt"
	"github.com/flant/server-access/util"
	"github.com/flant/server-access/vault"
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

type Config struct {
	vault.ServerAccessSettings
	vault.AuthdSettings
	DatabasePath string `json:"database"`
}

var (
	AppConfig Config
)

const (
	DefaultConfigFile   = "server-accessd.yaml"
	DefaultDatabasePath = "server-accessd.db"
)

func LoadConfig(fileName string) (Config, error) {
	if fileName == "" {
		fileName = DefaultConfigFile
	}
	cfg, err := LoadConfigFromFile(fileName)

	if err != nil {
		return Config{}, fmt.Errorf("load config '%s': %v", fileName, err)
	}

	if cfg == nil {
		cfg = &Config{}
	}

	cfg.ServerAccessSettings = vault.AssembleServerAccessSettings(cfg.ServerAccessSettings)
	cfg.AuthdSettings = vault.AssembleAuthdSettings(cfg.AuthdSettings)
	cfg.DatabasePath = util.FirstNonEmptyString(cfg.DatabasePath, os.Getenv("DATABASE"), DefaultDatabasePath)

	return *cfg, nil
}

func LoadConfigFromFile(fileName string) (*Config, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %s", fileName, err)
	}
	defer f.Close()

	return LoadConfigFromReader(f)
}

func LoadConfigFromReader(r io.Reader) (*Config, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, nil
	}

	cfg := new(Config)

	err = yaml.Unmarshal(buf.Bytes(), cfg)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal: %v", err)
	}

	return cfg, nil
}
