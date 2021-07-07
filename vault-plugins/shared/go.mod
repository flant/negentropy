module github.com/flant/negentropy/vault-plugins/shared

go 1.16

require (
	github.com/confluentinc/confluent-kafka-go v1.6.1
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/go-openapi/spec v0.20.3
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/validate v0.20.2
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	gopkg.in/square/go-jose.v2 v2.5.1
	sigs.k8s.io/yaml v1.2.0
)
