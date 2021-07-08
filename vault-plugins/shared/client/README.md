# Module for getting configured Vault API client

# Usage

Create controller and save it in backend 

`b.accessVaultController = client.NewVaultClientController(log.L)`

Add /configure_vault_access handle into backend

```
...
Paths: framework.PathAppend(
    ...
    
    []*framework.Path{
        client.PathConfigure(b.accessVaultController),
    },

    ...	
),
```


Add call `OnPeriodical` method in backend PeriodicalFunction

It is necessary otherwise vault access token will expire and client don't make requests.
The best way - add in top of function. Renew token is fast operation
```
PeriodicFunc: func(ctx context.Context, request *logical.Request) error {
	b.accessVaultController.OnPeriodical(ctx, request)
	...
},
```

Also call b.accessVaultController.Init(config.StorageView) in backend initialisation
It may return ErrNotSetConf error, but it is normal behavior.
Because, in first launch time, plugin has not saved setting

```
err = b.accessVaultController.Init(config.StorageView)
if err != nil && !errors.Is(err, client.ErrNotSetConf) {
	return err
}
```

For getting vault api client call 
```
apiClient, err := b.accessVaultController.APIClient()
```

Because client may not initialize before being receiving requests 
(and dynamical adding paths is [not allowed](https://github.com/hashicorp/vault/blob/f726f3ef163a71a02463bdb1428e69e3b69b6cd2/sdk/framework/backend.go#L40)) 
I recommend try to get client in top of handler
If client not get send error response else continue work

Full example see in `examples/` dir

For testing use commands

`build_and_start` - it starts vault

`make test` - it is runs initialisations plugins and roles 
and call two route witch call vault api with initialized client

Adding configuration example see in examples/Makefile in target `write_access_conf`

# Other

Generate certs for examples

```
echo '{"CN":"CA","key":{"algo":"rsa","size":2048}}' | cfssl gencert -initca - | cfssljson -bare ca -
echo '{"signing":{"default":{"expiry":"43800h","usages":["signing","key encipherment","server auth","client auth"]}}}' > ca-config.json
export ADDRESS=localhost,vault_host
export NAME=localhost
echo '{"CN":"'$NAME'","hosts":[""],"key":{"algo":"rsa","size":2048}}' | cfssl gencert -config=ca-config.json -ca=ca.pem -ca-key=ca-key.pem -hostname="$ADDRESS" - | cfssljson -bare $NAME
export ADDRESS=
export NAME=client
echo '{"CN":"'$NAME'","hosts":[""],"key":{"algo":"rsa","size":2048}}' | cfssl gencert -config=ca-config.json -ca=ca.pem -ca-key=ca-key.pem -hostname="$ADDRESS" - | cfssljson -bare $NAME
```
