package vault

import (
	"fmt"
	"github.com/flant/negentropy/authd"
	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/server-access/util"
	"github.com/hashicorp/vault/api"
	"os"
)

type AuthdSettings struct {
	Server     string `json:"server"`
	ServerType string `json:"serverType"`
	SocketPath string `json:"socketPath"`
}

const (
	DefaultVaultServer     = "auth.negentropy.flant.com"
	DefaultVaultServerType = v1.AuthServer
	DefaultAuthSocketPath  = "/run/authd/server-accessd.sock"
)

func AssembleAuthdSettings(settings AuthdSettings) AuthdSettings {
	return AuthdSettings{
		Server:     util.FirstNonEmptyString(settings.Server, os.Getenv("AUTHD_SERVER"), DefaultVaultServer),
		ServerType: util.FirstNonEmptyString(settings.ServerType, os.Getenv("AUTHD_SERVER_TYPE"), DefaultVaultServerType),
		SocketPath: util.FirstNonEmptyString(settings.SocketPath, os.Getenv("AUTHD_SOCKET"), DefaultAuthSocketPath),
	}
}

func ClientFromAuthd(settings AuthdSettings) (*api.Client, error) {
	authdClient := authd.NewAuthdClient(settings.SocketPath)

	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("*", map[string]string{})).
		WithServerType(settings.ServerType)

	err := authdClient.OpenVaultSession(req)
	if err != nil {
		return nil, fmt.Errorf("open vault session with authd: %v", err)
	}

	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("get vault client from authd: %v", err)
	}

	return vaultClient, nil
}
