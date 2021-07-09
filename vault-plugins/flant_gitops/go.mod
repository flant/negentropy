module github.com/flant/negentropy/vault-plugins/flant_gitops

go 1.16

require (
	github.com/docker/docker v1.4.2-0.20200319182547-c7ad2b866182
	github.com/flant/negentropy/vault-plugins/shared v0.0.0-20210708154747-5592c1b9eceb
	github.com/go-git/go-git/v5 v5.3.0
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v0.16.1
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/vault/api v1.1.0
	github.com/hashicorp/vault/sdk v0.2.0
	github.com/werf/logboek v0.5.4
	github.com/werf/vault-plugin-secrets-trdl v0.0.0-20210707133059-d7d1d5e42e5b
)

replace github.com/theupdateframework/go-tuf => github.com/werf/third-party-go-tuf v0.0.0-20210420212757-8e2932fb01f2
