module github.com/flant/negentropy/vault-plugins/e2e

go 1.16

require (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.1
	github.com/tidwall/gjson v1.8.1
)

replace (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../flant_iam
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
)
