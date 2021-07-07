package config

import (
	"fmt"
	"sigs.k8s.io/yaml"
)

const AuthdConfigKind = "AuthdConfig"

/*
apiVersion: authd.example.com/v1alpha1
kind: AuthdConfig
jwtPath: /var/lib/authd.jwt
servers:
- type: RootSource
  domain: root-source.auth.example.com
- type: Auth
  domain: auth.example.com
  allowRedirects:
  - *.auth.example.com
- type: Auth
  domain: auth2.example.com
  allowRedirects:
  - *.auth2.example.com
*/
type AuthdConfig struct {
	Metadata Metadata
	// Versioned configs.
	cfgV1 *AuthdConfigV1
}

func (a *AuthdConfig) GetJWTPath() string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.JwtPath
	}
	return ""
}

func (a *AuthdConfig) GetServers() []Server {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.Servers
	}
	return nil
}

func (a *AuthdConfig) GetDefaultSocketDirectory() string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.DefaultSocketDirectory
	}
	return ""
}

type Server struct {
	Type           string   `json:"type"`
	Domain         string   `json:"domain"`
	AllowRedirects []string `json:"allowedRedirects,omitempty"`
}

type AuthdConfigV1 struct {
	JwtPath                string   `json:"jwtPath"`
	DefaultSocketDirectory string   `json:"defaultSocketDirectory"`
	Servers                []Server `json:"servers"`
}

func (c *AuthdConfig) Load(metadata Metadata, data []byte) error {
	var err error

	switch metadata.Version {
	case "v1":
		c.Metadata = metadata
		c.cfgV1, err = c.LoadV1(data)
	default:
		err = fmt.Errorf("version '%s' is not supported", metadata.ApiVersion())
	}
	return err
}

func (c *AuthdConfig) LoadV1(data []byte) (*AuthdConfigV1, error) {
	var cfgV1 = new(AuthdConfigV1)

	err := yaml.Unmarshal(data, cfgV1)
	if err != nil {
		return nil, err
	}

	return cfgV1, nil
}

const AuthdSocketConfigKind = "AuthdSocketConfig"

/*
apiVersion: authd.negentropy.flant.com/v1alpha1
kind: AuthdSocketConfig
path: /var/run/my.sock
user: root
group: root
mode: 0600
allowedServerTypes: [RootSource, Auth]
allowedPolicies:
- policy: iam.view
- policy: iam.edit
- policy: server.ssh.*
*/
type AuthdSocketConfig struct {
	Metadata Metadata

	cfgV1 *AuthdSocketConfigV1
}

func (a *AuthdSocketConfig) GetPath() string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.Path
	}
	return ""
}

func (a *AuthdSocketConfig) GetUser() string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.User
	}
	return ""
}

func (a *AuthdSocketConfig) GetGroup() string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.Group
	}
	return ""
}

func (a *AuthdSocketConfig) GetMode() int {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.Mode
	}
	return 0
}

func (a *AuthdSocketConfig) GetAllowedServerTypes() []string {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.AllowedServerTypes
	}
	return nil
}

func (a *AuthdSocketConfig) GetAllowedPolicies() []AllowedPolicy {
	if a.Metadata.Version == "v1" {
		return a.cfgV1.AllowedPolicies
	}
	return nil
}

type AllowedPolicy struct {
	Policy string `json:"policy"`
}

type AuthdSocketConfigV1 struct {
	Path               string          `json:"path"`
	User               string          `json:"user"`
	Group              string          `json:"group"`
	Mode               int             `json:"mode"`
	AllowedServerTypes []string        `json:"allowedServerTypes"`
	AllowedPolicies    []AllowedPolicy `json:"allowedPolicy"`
}

func (c *AuthdSocketConfig) Load(metadata Metadata, data []byte) error {
	var err error
	switch metadata.Version {
	case "v1":
		c.Metadata = metadata
		c.cfgV1, err = c.LoadV1(data)
	default:
		err = fmt.Errorf("version '%s' is not supported", metadata.ApiVersion())
	}
	return err
}

func (c *AuthdSocketConfig) LoadV1(data []byte) (*AuthdSocketConfigV1, error) {
	var cfgV1 = new(AuthdSocketConfigV1)

	err := yaml.Unmarshal(data, cfgV1)
	if err != nil {
		return nil, err
	}

	//// Convert socket file mode.
	//mode, err := strconv.ParseUint(cfgV1.Mode, 8, 32)
	//if err != nil {
	//	return err
	//}
	//c.Mode = uint32(mode)
	return cfgV1, nil
}
