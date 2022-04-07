module probs

go 1.16

require (
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/confluentinc/confluent-kafka-go v1.7.0
	github.com/fatih/color v1.13.0 // indirect
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/hashicorp/go-hclog v1.0.0
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pkg/profile v1.6.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20211111083644-e5c967477495 // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.44.0 // indirect
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared

replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.1 => ../vault-plugins/flant_iam
