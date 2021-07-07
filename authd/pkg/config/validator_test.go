package config

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Validate(t *testing.T) {

	var vu *VersionedUntyped
	var err error

	var tests = []struct {
		name       string
		configText string
		fn         func()
	}{
		{
			"Wrong metadata",
			`
apiVersion: v12
kind: AuthdConfig
`,
			func() {
				require.Error(t, err, "")
			},
		},
		{
			"AuthdConfig/v1 without error",
			`
apiVersion: v1
kind: AuthdConfig
jwtPath: /var/run/authd.sock
servers:
  - type: RootSource
    domain: root-source.negentropy.example.com
  - type: Auth
    domain: auth.negentropy.example.com
    allowRedirects: 
    - "*.auth.negentropy.example.com"
`,
			func() {
				require.NoError(t, err, "")
			},
		},
		{
			"AuthdConfig/v1 error additional fields",
			`
apiVersion: v1
kind: AuthdConfig
jwtPaths: /var/run/authd.sock
serverz:
  - type: RootSource
    domain: root-source.negentropy.example.com
  - type: Auth
    domain: auth.negentropy.example.com
    allowRedirects: 
    - "*.auth.negentropy.example.com"
`,
			func() {
				require.Error(t, err, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vu = prepareConfigObj(t, tt.configText)
			s := GetSchema(vu.Metadata.Kind + "/" + vu.Metadata.Version)
			err = ValidateConfig(vu.Object(), s, "root")
			//t.Logf("expected multierror was: %v", err)
			tt.fn()
		})
	}
}

func prepareConfigObj(t *testing.T, input string) *VersionedUntyped {
	vu := new(VersionedUntyped)
	err := vu.DetectMetadata([]byte(input))

	require.NoError(t, err, "apiVersion and kind should be detected")
	return vu
}
