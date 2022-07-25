module github.com/flant/negentropy/vault-plugins/flant_iam_auth

go 1.17

require (
	github.com/GehirnInc/crypt v0.0.0-20200316065508-bb7000b8a962
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/confluentinc/confluent-kafka-go v1.9.1
	github.com/coreos/go-oidc v2.2.1+incompatible // indirect
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/go-test/deep v1.0.8
	github.com/gojuno/minimock/v3 v3.0.10
	github.com/hashicorp/cap v0.2.0
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.2.1
	github.com/hashicorp/go-memdb v1.3.3
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/vault/api v1.7.2
	github.com/hashicorp/vault/sdk v0.5.3
	github.com/mitchellh/pointerstructure v1.2.1
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.0
	github.com/open-policy-agent/opa v0.42.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/ryanuber/go-glob v1.0.0
	github.com/stretchr/testify v1.8.0
	github.com/tidwall/gjson v1.14.1
	golang.org/x/oauth2 v0.0.0-20220722155238-128564f6959c
	gopkg.in/square/go-jose.v2 v2.6.0
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.24.3
)

replace github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../shared
replace github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../flant_iam
