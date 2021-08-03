module github.com/flant/negentropy/vault-plugins/flant_iam

go 1.16

require (
	github.com/confluentinc/confluent-kafka-go v1.6.1
	github.com/fatih/color v1.10.0 // indirect
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/sethvargo/go-password v0.2.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.8.1
	k8s.io/apimachinery v0.21.2
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
