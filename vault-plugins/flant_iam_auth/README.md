# build 

```
../dev-build.sh
```

# prepare for tests

```
../dev-build.sh && \
  docker-compose down && \
  ../dev-start.sh up && \
  ../dev-start.sh connect_plugins && \
  echo "Permanen root token:" && \
  cat /tmp/token_root && \
  echo ""
```

# tests
Also all tests can run from Goland IDE
 
### integration and unit
Run vault (see above)

```
VAULT_ADDR=http://127.0.0.1:8200 go test -v ./...
```

### e2e
- Run vault (see above)
- `cd ../e2e`  
- Unskip tests in `flow` directory (we skip it because it is not setup in ci)
- 
 ```
TEST_VAULT_SECOND_TOKEN=$(cat /tmp/token_root) go test -v ./...
```

it is is slow because we sleep for kafka sync

# logs

Pretty logs for vault

```
docker logs -f "$(docker ps | grep vault:1.7 | cut -d ' ' -f 1)" 2>&1 | jq -R 'fromjson? | "[" + .["@module"] + "] - [" + .["@level"] + "] - " + .["@message"]'
```