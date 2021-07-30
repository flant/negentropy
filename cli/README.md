# cli

flant CLI utility

## Build

- git clone git@github.com:flant/negentropy.git
- cd cli
- GO111MODULE=on go build -o /OPT/cli main.go

## Deployment

- /OPT/cli should be delivered to every client PC
- cli needs authd at client PC
- cli should be configured to connect to authd  
  (configuration is under construction now)