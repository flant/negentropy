module github.com/flant/negentropy/server-access

go 1.17

require (
	github.com/flant/negentropy/authd v0.0.0
	github.com/flant/negentropy/cli v0.0.0
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/gojuno/minimock/v3 v3.0.10
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/hashicorp/vault/api v1.7.2
	github.com/jmoiron/sqlx v1.3.5
	github.com/mattn/go-sqlite3 v1.14.14
	github.com/otiai10/copy v1.7.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.0
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/flant/negentropy/authd v0.0.0 => ../authd
	github.com/flant/negentropy/cli v0.0.0 => ../cli
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../vault-plugins/flant_iam
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0 => ../vault-plugins/flant_iam_auth
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
)
