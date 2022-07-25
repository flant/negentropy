module github.com/flant/negentropy/vault-plugins/flant_iam

go 1.16

require (
	github.com/confluentinc/confluent-kafka-go v1.9.1
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v1.2.1
	github.com/hashicorp/go-memdb v1.3.3
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/vault/api v1.7.2
	github.com/hashicorp/vault/sdk v0.5.3
	github.com/hashicorp/yamux v0.1.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.0
	github.com/sethvargo/go-password v0.2.0
	github.com/stretchr/testify v1.8.0
	github.com/tidwall/gjson v1.14.1
	k8s.io/apimachinery v0.24.3
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
