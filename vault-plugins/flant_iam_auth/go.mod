module github.com/flant/negentropy/vault-plugins/flant_iam_auth

go 1.16

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/confluentinc/confluent-kafka-go v1.6.1
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/getkin/kin-openapi v0.66.0 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/strfmt v0.20.1 // indirect
	github.com/go-openapi/validate v0.20.2 // indirect
	github.com/go-test/deep v1.0.2
	github.com/hashicorp/cap v0.0.0-20210204173447-5fcddadbf7c7
	github.com/hashicorp/errwrap v1.0.0
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-memdb v1.3.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/vault/api v1.0.5-0.20200519221902-385fac77e20f
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/mitchellh/pointerstructure v1.0.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/ryanuber/go-glob v1.0.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/square/go-jose.v2 v2.5.1
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared

replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../flant_iam
