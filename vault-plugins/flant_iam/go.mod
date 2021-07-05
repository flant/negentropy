module github.com/flant/negentropy/vault-plugins/flant_iam

go 1.15

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/confluentinc/confluent-kafka-go v1.6.1
	github.com/fatih/color v1.10.0 // indirect
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
