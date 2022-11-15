# flant_gitops

### What flant_gitops does?

flant_gitops plugin watches for new commits in the git repository and runs specified user command periodically.

- This command will be executed strictly in the docker container using specified docker image.
- The only way to pass additional data for this command is to use data from the commit of the git repository.
- User command will be executed only when a new commit have arrived into the specified branch.
- flant_gitops optionally verifies that a new commit has been signed with the required number of trusted pgp keys before running user command.
- flant_gitops could pre execute an arbitrary number of requests into some vault server before running a command specified by the user.

### How to use it?

#### Main configuration (required)

```
vault write flant_gitops/configure PARAMS
```

The main params are: git repository address to watch to, git repository branch to watch for new commits, required number of verified pgp signatures on commit etc. Get full params list with descriptions with the following command:

```
vault path-help flant-gitops/configure/
```

It is required to configure flant_gitops to enable periodic running of user command.

#### Vault requests (optional)

All configured requests are performed in the wrapped mode: flant_gitops obtains a token for each named request, then passes these tokens into the container command using environment variables named as `$VAULT_REQUEST_TOKEN_<VAULT_REQUEST_NAME>`. It is possible to get request responses for each request using these token by calling an unwrap operation (`vault write sys/wrapping/unwrap token=XXX` for example) from inside container.

It is required to configure vault access into this vault server. It could be the same vault server, which runs flant_gitops plugin or it could be totally different vault server.

##### Configure vault access

1. Enable `approle` in the target vault:

    ```
    vault auth enable approle
    ```

2. Create a policy in the target vault:

    ```
    vault policy write mypolicy mypolicy.hcl
    ```

    Where `mypolicy.hcl` could contain something like:

    ```
    path "*" {
        capabilities = ["create", "read", "update", "delete", "list"]
    }
    ```

3. Create a role in the target vault:

    ```
    vault write auth/approle/role/myrole secret_id_ttl=30m token_ttl=90s token_policies=mypolicy
    ```

    Get and remember myrole id:

    ```
    vault read -format=json auth/approle/role/myrole/role-id | jq -r '.data.role_id'
    ```

    Also get and remember myrole secret_id:

    ```
    vault write -format=json -f auth/approle/role/myrole/secret-id | jq -r '.data.secret_id'
    ```

4. Configure vault access in the flant_gitops plugin:

    ```
    vault write flant_gitops/configure_vault_access \
        vault_addr=TARGET_VAULT_ADDR \
        vault_tls_server_name=TARGET_VAULT_TLS_SERVER_NAME \
        role_name="myrole" \
		secret_id_ttl="120m" \
		approle_mount_point="auth/approle" \
        secret_id=MYROLE_SECRET_ID \
        role_id=MYROLE_ID
		vault_cacert=TARGET_VAULT_CACERT_PEM
    ```

##### Configure vault request

```
vault write flant_gitops/configure/vault_request/VAULT_REQUEST_NAME PARAMS
```

`VAULT_REQUEST_NAME` — arbitrary request name. The result of request will be available in the container command by environment variable `VAULT_REQUEST_TOKEN_<VAULT_REQUEST_NAME>` (vault request name is capitalized and snake-cased).

See following help command for other params:

```
vault path-help flant_gitops/configure/vault_request/VAULT_REQUEST_NAME
```

### Examples

Start vault dev server in one terminal:

```
make
```

Enable flant_gitops plugin, load supplement fixture data and watch log in another terminal:

```
make enable
```


### Kubernetes

```
curl -s https://raw.githubusercontent.com/flant/negentropy/main/bootstrap-kube.sh| bash
```