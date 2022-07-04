module github.com/flant/negentropy/kafka-consumer

go 1.16

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/confluentinc/confluent-kafka-go v1.6.1
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v1.2.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.7.1
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
