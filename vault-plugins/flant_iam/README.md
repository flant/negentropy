# flant_iam

The IAM module.

## Development

Go to the path above, so you are in `vault-plugins` dir.

Exec

```sh
flant_iam/tests/start.sh
```

To rebuild a module, run

```sh
docker exec gobuild /gobuild.sh flant_iam
```


To mount updated module, run

```sh
docker exec dev-vault /remount.sh flant_iam
```


Run tests
```sh
cd flant_iam
make deps
make test
```