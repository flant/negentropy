# server-accessd

The daemon for servers to have synced users and ssh access from vault entities. Use authd to connect to Vault. Use
server-access plugin to get users and their certificates.

## build

- git clone git@github.com:flant/negentropy.git
- cd server-access/flant-server-accessd
- GO111MODULE=on go build -o server-accessd ./cmd/main.go

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