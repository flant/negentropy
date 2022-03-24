# server-accessd

The daemon for servers to have synced users and ssh access from vault entities. Use authd to connect to Vault. Use
server-access plugin to get users and their certificates.

## build

- git clone git@github.com:flant/negentropy.git
- cd negentropy/server-access
- GO111MODULE=on go build -o server-accessd .flant-server-accessd/cmd/main.go

## deploy & config

- binary should be delivered to every server
- at the server should be run authd
- binary should be run as a service

```
example of serveraccessd.service
--------------------------------
[Unit]
Description=Negentropy server controller
After=network.target

[Service]
PrivateTmp=true
Type=exec
PIDFile=/var/run/serveraccessd/%i.pid
ExecStart=/opt/serveraccessd/bin/serveraccessd
Environment=SERVER_ACCESSD_CONF=/opt/serveraccessd/config.yaml

[Install]
WantedBy=multi-user.target
```

- binary needs config at path specified in $SERVER_ACCESSD_CONF

```
example of opt/serveraccessd/config.yaml
----------------------------------------
tenant: 6156a009-9263-4212-b1d3-55122317230b
project: 72a53827-101c-4ae0-a134-121aea1493de
server: 727d8e80-3be9-4f73-a36b-3bbebaff0f47
database: /opt/serveraccessd/server-accessd.db
socketPath: /run/authd.sock
```

*tenant*, *project*, *server* are uuids from negentropy system  
*database* is a path to a db-file which is common for both server-accessd and server-access-nss, DON'T CHANGE IT      
*socketPath* should be syncronized with config of *authd*

Restart service after change configuration

## debug running daemon

from negentropy folder:

- build run and prepare system

 ```
  ./build.sh # build components
  ./build.sh vault --force # for forced rebuild vault
  ./start.sh e2e # run system
  ./run-e2e-tests.sh # run tests for creating user and other staff
  ```

- copy from test-server docker container multipass-jwt file /opt/authd/client-jwt to authd/dev/secret/authd.jwt :  
  ```docker cp  test-server:/opt/authd/server-jwt authd/dev/secret/authd.jwt```

- copy from test-server docker container file /opt/server-access/config.yaml to
  server-access/flant-server-accessd/dev/config.yaml  
  ```docker cp  test-server:/opt/server-access/config.yaml  server-access/flant-server-accessd/dev/config.yaml```

- edit *server-access/flant-server-accessd/dev/config.yaml*  
  replace:
  ``` 
  database: /opt/serveraccessd/server-accessd.db
  authdSocketPath: /run/sock1.sock 
  ```     

  for
  ```
  database: server-accessd.db
  authdSocketPath: ../authd/dev/run/sock1.sock
  ```

from authd folder

- run authd:  
  ```go run cmd/authd/main.go --conf-dir=dev/conf```

from server-access folder

- run serveraccessd:    
  ```export SERVER_ACCESSD_CONF=./flant-server-accessd/dev/config.yaml; go run ./flant-server-accessd/cmd```

