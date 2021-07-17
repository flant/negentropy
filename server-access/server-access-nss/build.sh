#!/usr/bin/env bash

ldflags="-s -w"
#ldflags="-s -w -X 'main.UserDatabasePath=/src/users.db'"

echo "Compile libnss_flantauth.so.2 ..."

CGO_CFLAGS="-g -O2 -D __LIB_NSS_NAME=flantauth" \
go build \
   -ldflags="${ldflags}" \
   -buildmode=c-shared \
   -o libnss_flantauth.so.2

if [[ $1 == "install" ]] ; then
  echo "Copy libnss_flantauth.so.2 to /lib/x86_64-linux-gnu/libnss_flantauth.so.2"
  cp libnss_flantauth.so.2 /lib/x86_64-linux-gnu/
fi
