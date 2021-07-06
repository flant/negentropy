module github.com/flant/negentropy/explugin

go 1.16

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../../../shared

replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.1 => ../../../flant_iam

require (
	github.com/confluentinc/confluent-kafka-go v1.7.0
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.1
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-hclog v0.16.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.0
)
