# Negentropy

Multi tenant enterprise grade IAM implementation based on a geo-distributed layered Vault installation. Currently under
active development of initial version.

# Build components for e2e tests

needs:

1) linux or macos system (only intel processors, for apple_silicone using needs some changes in scripts)
2) docker

```shell
./build.sh
```

possible options for build:

```shell
./build.sh plugins # build separate plugins
```

```shell
./build.sh authd  # builds authd component
```

```shell
./build.sh cli # builds cli utility
```

```shell
./build.sh server-accessd # builds server-accessd component
```

```shell
./build.sh nss # builds nss component
```

```shell
./build.sh oidc-mock # builds oidc-mock for e2e tests purposes only 
```

```shell
./build.sh vault          # builds complete vault with plugins onboard
./build.sh vault --force  # builds complete vault with plugins onboard (use after first build)  
```

# Run environment for e2e tests

There are three possible modes of test and stage environment:

1) One vault in dev mode, negentropy plugins are aside (DEV mode)
2) Vaults with negentropy plugins onbоard (NORMAL mode)
3) Vaults with negentropy plugins onbоard, run under delve debugger (DEBUG mode)

## DEV mode:

```shell
./start.sh --dev
```

runs one vault at docker-container, uses separate plugin binaries, placed at vault-plugins/build

## NORMAL mode:

```shell
./start.sh
```

runs several vaults at docker-containers, uses complete vault binary with negentropy plugins onboard, placed at
infra/common/vault/vault/bin

## DEBUG mode

```shell
./start.sh --debug
```

runs several vaults at docker-containers, each docker run under delve debugger server uses complete vault binary with
negentropy plugins onboard, placed at infra/common/vault/vault/bin, need connection delve-client debuggers to localhost:
2345 and localhost:2346 (see docker/docker-compose.debug.yml)

## General components in other docker containers

1) Zookepper, Kafka used to save data and communicate plugins.
2) Kafdrop used to study Kafka
3) test-server used as a sample of server under negentropy access control
4) test-client used as a sample of user PC, accessing servers under negentropy access control
5) oidc-mock provide mock of oidc-provider for tests

## start.sh matter

1) run all components containers
2) configure negentropy plugins
3) export data for running tests and unsealing vaults

# Seal/Unseal vaults

For stopping in NORMAL and DEBUG mode:

```shell
./stop.sh
```

For stopping in DEV mode:

```shell
./stop-dev.sh
```

For restore:

1) run vault container(s), run delve clients for DEBUG mode
2) execute:
```shell
./unseal.sh
```

# E2E tests:

```shell
./run-e2e-tests.sh
```