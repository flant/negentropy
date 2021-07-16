#!/usr/bin/env bash

echo "Comile and copy to /lib/x86_64-linux-gnu/libnss_flantauth.so.2"
CGO_CFLAGS="-g -O2 -D __LIB_NSS_NAME=flantauth" go build  --buildmode=c-shared -o /lib/x86_64-linux-gnu/libnss_flantauth.so.2
