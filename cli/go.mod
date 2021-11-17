module github.com/flant/negentropy/cli

go 1.16

require (
	github.com/flant/negentropy/authd v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0
	github.com/google/uuid v1.2.0
	github.com/hashicorp/vault/api v1.1.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210415154028-4f45737414dc
	gopkg.in/square/go-jose.v2 v2.5.1
)

replace (
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../vault-plugins/flant_iam
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0 => ../vault-plugins/flant_iam_auth
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
        github.com/flant/negentropy/authd v0.0.0 => ../authd
)
