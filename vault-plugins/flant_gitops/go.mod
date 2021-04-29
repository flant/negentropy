module github.com/flant/negentropy/vault-plugins/flant_gitops

go 1.16

require (
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v0.16.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/werf/logboek v0.5.4
	github.com/werf/vault-plugin-secrets-trdl v0.0.0-20210428180238-a92bb5c827ac
	golang.org/x/text v0.3.5 // indirect
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210420212757-8e2932fb01f2
