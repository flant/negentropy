# authd

authd is a proxy to use the pool of Vault servers. It's main goal is to use JWT to open Vault sessions for other clients.

## Development

`dev` directory contains a simple setup to test authd:
- script to startup vault dev server: it creates an entity and entity_alias, open session and issues a JWT. Then config auth/jwt and ssh plugins for client_test.go.
- script to refresh JWT
- script to emulate redirects within pool: run nginx that redirects authd requests to a dev server
- configuration with one socket

## TODO

1. OpenAPI validation for config. Validator is done, it needs to be added to authd daemon.
2. Session token refresher for client library.
3. ~~URL and arguments in configuration for Vault requests (auth/myjwt/login is hardcoded).~~
4. ~~Rename policy to role~~, do role checks in LoginHandler.
5. In-memory Vault instance for testing.
6. ~~Apply socket file permissions.~~
7. Support for penging login.


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
