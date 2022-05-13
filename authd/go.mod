module github.com/flant/negentropy/authd

go 1.15

require (
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/go-chi/chi/v5 v5.0.2
	github.com/go-openapi/spec v0.20.3
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.14
	github.com/go-openapi/validate v0.20.2
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
