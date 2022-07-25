module github.com/flant/negentropy/authd

go 1.17

require (
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-openapi/spec v0.20.6
	github.com/go-openapi/strfmt v0.21.3
	github.com/go-openapi/swag v0.21.1
	github.com/go-openapi/validate v0.22.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/vault/api v1.7.2
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.0
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	sigs.k8s.io/yaml v1.3.0
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
