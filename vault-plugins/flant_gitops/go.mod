module github.com/flant/negentropy/vault-plugins/flant_gitops

go 1.16

require (
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/docker/docker v20.10.17+incompatible
	github.com/fatih/structs v1.1.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/hashicorp/go-hclog v1.2.1
	github.com/hashicorp/vault v1.11.1
	github.com/hashicorp/vault/api v1.7.2
	github.com/hashicorp/vault/sdk v0.5.3
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.8.0
	github.com/tencentcloud/tencentcloud-sdk-go v3.0.171+incompatible // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/werf/logboek v0.5.4
	github.com/werf/vault-plugin-secrets-trdl v0.0.0-20210824164229-ed847e15b393
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
