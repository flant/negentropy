module github.com/flant/negentropy/e2e

go 1.16

require (
	github.com/docker/docker v20.10.17+incompatible
	github.com/flant/negentropy/authd v0.0.0
	github.com/flant/negentropy/cli v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v1.2.1
	github.com/hashicorp/vault/api v1.7.2
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.0
	github.com/tidwall/gjson v1.14.1
	gopkg.in/square/go-jose.v2 v2.6.0
)

replace (
	github.com/flant/negentropy/authd v0.0.0 => ../authd
	github.com/flant/negentropy/cli v0.0.0 => ../cli
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../vault-plugins/flant_iam
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0 => ../vault-plugins/flant_iam_auth
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
)
