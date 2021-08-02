module github.com/flant/negentropy/e2e

go 1.16

require (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.1
	github.com/tidwall/gjson v1.8.1
	gopkg.in/square/go-jose.v2 v2.5.1
)

replace (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../vault-plugins/flant_iam
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0 => ../vault-plugins/flant_iam_auth
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
)
