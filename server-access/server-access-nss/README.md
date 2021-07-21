# libnss_flantauth

A library to store users in an sqlite db.

## Build

```bash
make build
```

#### build and install into /lib/x86_64-linux-gnu/

```bash
make build install
```

## Installation

Put libnss_flantauth.so.2 into `/lib/x86_64-linux-gnu` or `/usr/lib64` and add `flantauth` service to `passwd`, `group` and `shadow` databases in nsswitch.conf:

```
passwd:   files flantauth
group:    files flantauth
shadow:   files flantauth
```

## Tet

You can launch integration test by run:

```
make test
```

it will build libnss_flantauth.so.2 in `out` directory and run different versions of Debian and Ubuntu to test against users.db file.

