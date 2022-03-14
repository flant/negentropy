module github.com/flant/negentropy/vault-plugins/flant_gitops

go 1.16

require (
	github.com/docker/docker v17.12.0-ce-rc1.0.20200309214505-aa6a9891b09c+incompatible
	github.com/fatih/structs v1.1.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/go-git/go-git/v5 v5.3.0
	github.com/hashicorp/go-hclog v0.16.1
	github.com/hashicorp/vault v1.7.3
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.1-0.20210614231108-a35199734e5f
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.8.1 // indirect
	github.com/werf/logboek v0.5.4
	github.com/werf/vault-plugin-secrets-trdl v0.0.0-20210823182616-23a91b817bc8
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
