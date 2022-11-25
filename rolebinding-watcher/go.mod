module github.com/flant/negentropy/rolebinding-watcher

go 1.17

require (
	github.com/flant/negentropy/kafka-consumer v0.0.1
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0
	github.com/flant/negentropy/vault-plugins/shared v0.0.1
	github.com/hashicorp/go-hclog v1.2.1
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.10.1
)

require (
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/confluentinc/confluent-kafka-go v1.9.1 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-kms-wrapping/entropy v0.1.0 // indirect
	github.com/hashicorp/go-memdb v1.3.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.4 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/hashicorp/vault/sdk v0.5.3 // indirect
	github.com/hashicorp/yamux v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.4.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220808131553-a91ffa7f803e // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	github.com/flant/negentropy/e2e v0.0.0 => ../e2e
	github.com/flant/negentropy/kafka-consumer v0.0.1 => ../kafka-consumer
	github.com/flant/negentropy/vault-plugins/flant_iam v0.0.0 => ../vault-plugins/flant_iam
	github.com/flant/negentropy/vault-plugins/flant_iam_auth v0.0.0 => ../vault-plugins/flant_iam_auth
	github.com/flant/negentropy/vault-plugins/shared v0.0.1 => ../vault-plugins/shared
)
