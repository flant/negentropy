module probs

go 1.16

require (
	github.com/confluentinc/confluent-kafka-go v1.7.0
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/docker/docker v20.10.8+incompatible
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.1
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v1.0.0
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/open-policy-agent/opa v0.38.0
	github.com/pkg/profile v1.6.0
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.0
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared

replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.1 => ../vault-plugins/flant_iam
