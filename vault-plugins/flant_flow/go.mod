module github.com/flant/negentropy/vault-plugins/flant_flow

go 1.16

require (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.8.1
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../flant_iam