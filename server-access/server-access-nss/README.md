# libnss_flantauth

A library to store users in an sqlite db.

## Build

```
CGO_CFLAGS="-g -O2 -D __LIB_NSS_NAME=flantauth" \
go build \
   -ldflags="-s -w" \
   -buildmode=c-shared \
   -o libnss_flantauth.so.2
```

You can change a database path using -X flag:

```
-ldflags="-s -w -X 'main.UserDatabasePath=/etc/access/users.db'"
```


## Installation

Put libnss_flantauth.so.2 into `/lib/x86_64-linux-gnu` or `/usr/lib64` and add `flantauth` service to `passwd`, `group` and `shadow` databases in nsswitch.conf:

```
passwd:   flantauth files
group:   flantauth files
shadow:   flantauth files
```

## Example

This repository has `example` folder with simple database and a script:

```
cd example
./run.sh
```

`run.sh` will build libnss_flantauth.so.2 in `out` directory and run different versions of Debian and Ubuntu to test against users.db file.

