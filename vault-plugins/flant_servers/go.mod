module github.com/flant/negentropy/vault-plugins/flant_servers

go 1.16

replace github.com/flant/negentropy/vault-plugins/shared v0.0.0 => ../shared

replace github.com/flant/negentropy/vault-plugins/flant_iam => ../flant_iam

require (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0-20210429072305-24eb8fd49da4
	github.com/flant/negentropy/vault-plugins/shared v0.0.0
	github.com/hashicorp/go-hclog v0.15.0
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/vault v1.7.1
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.1-0.20210419223509-b296e151b5b3
	github.com/stretchr/testify v1.7.0
)
