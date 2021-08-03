# authd

authd is a proxy to use the pool of Vault servers. It's main goal is to use JWT to open Vault sessions for other
clients.

## build

- git clone git@github.com:flant/negentropy.git
- cd negentropy/authd
- GO111MODULE=on go build -o /OPT/authd/bin/authd cmd/authd/main.go

## deploy & config

- binary should be delivered to every client PC & server
- binary should be run as a service

```
example of authd.service:
-------------------------
[Unit]
Description=Daemon to authenticate in Negentropy
After=network.target

[Service]
PrivateTmp=true
Type=exec
PIDFile=/var/run/authd/%i.pid
ExecStart=/opt/authd/bin/authd

[Install]
WantedBy=multi-user.target
```

- binary needs configs in /etc/flant/negentropy/authd-conf.d directory
  examples of yaml configs are in authd/dev/conf  
  main.yaml provide connection to vault  
  sock1.yaml etc - are configs for connections to authd
- binary need valid jwt token at the path specified in main.yaml[jwtPath]  
  the way to get example of jwt is available through scripts in authd/dev
- restart authd after change config or jwt

## Development

`dev` directory contains a simple setup to test authd:

- script to startup vault dev server: it creates an entity and entity_alias, open session and issues a JWT. Then config
  auth/jwt and ssh plugins for client_test.go.
- script to refresh JWT
- script to emulate redirects within pool: run nginx that redirects authd requests to a dev server
- configuration with one socket

## TODO

1. OpenAPI validation for config. Validator is done, it needs to be added to authd daemon.
2. Session token refresher for client library.
3. ~~URL and arguments in configuration for Vault requests (auth/myjwt/login is hardcoded).~~
4. ~~Rename policy to role~~, do checks in LoginHandler.
5. In-memory Vault instance for testing.
6. Apply socket file permissions.


## Links

https://www.vaultproject.io/docs/secrets/identity


https://www.vaultproject.io/api-docs/secret/identity/tokens
https://www.vaultproject.io/api-docs/secret/identity/tokens#client_id
https://www.vaultproject.io/api-docs/secret/identity/entity#read-entity-by-id
https://www.vaultproject.io/docs/concepts/lease
https://www.vaultproject.io/api-docs/secret/identity/entity-alias
https://www.vaultproject.io/api/system/auth#list-auth-methods
https://www.vaultproject.io/api-docs/auth/token#create-token
https://www.vaultproject.io/api/auth/token
https://www.vaultproject.io/docs/auth/token
https://www.vaultproject.io/api-docs/auth/jwt
