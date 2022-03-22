# cli

flant CLI utility

## Build

- git clone git@github.com:flant/negentropy.git
- cd negentropy/cli
- GO111MODULE=on go build -o /OPT/cli main.go

## Deployment

- /OPT/cli should be delivered to every client PC
- cli needs authd at client PC
- cli should be configured to connect to authd  
  (configuration is under construction now)

## Running for debug

from root folder:

- build run and prepare system

 ```
  ./build.sh # build components
  ./build.sh vault --force # for forced rebuild vault
  ./start.sh e2e # run system
  ./run-e2e-tests.sh # run tests for creating user and other staff
  ```

- copy from test-client docker container multipass-jwt file /opt/authd/client-jwt to authd/dev/secret/authd.jwt :
  ```docker cp  test-client:/opt/authd/client-jwt authd/dev/secret/authd.jwt```
  from authd folder
- run authd:
  ```go run cmd/authd/main.go --conf-dir=dev/conf```
  from cli folder
- run cli:
  ```export CACHE_PATH=cache; export AUTHD_SOCKET_PATH=../authd/dev/run/sock1.sock; go run cmd/cli/main.go get tenant --all-tenants```
  