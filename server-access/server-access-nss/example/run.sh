#!/usr/bin/env bash

set -e

SCRIPTDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit 2 ; pwd -P )"

ubuntuList=
ubuntuList="ubuntu:16.04 ubuntu:18.04 ubuntu:20.04 ubuntu:21.04"

alpineList=
#alpineList="alpine:3.12 alpine:3.13 alpine:3.14"

debianList=
debianList="debian:7 debian:8 debian:9 debian:10"

centosList=
centosList="centos:6 centos:7 centos:8"

mkdir -p "$SCRIPTDIR"/out
# build universal .so in buster based image.
if [[ ! -f $SCRIPTDIR/../lib/libnss_flantauth.so.2 ]] ; then
  echo "libnss_flantauth.so.2 does not exist. Run make build first"
  exit 1;
fi

cp "$SCRIPTDIR"/../lib/libnss_flantauth.so.2 "$SCRIPTDIR"/out
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  ldd ./out/libnss_flantauth.so.2
fi


# Run in Debian and Ubuntu.
for image in $debianList $ubuntuList ; do
  echo Run example in $image
  cat <<EOF | docker run \
    -w /example \
    -v "$SCRIPTDIR":/example \
    --rm -i $image sh -s $image

image=$1

cp ./out/libnss_flantauth.so.2 /lib/x86_64-linux-gnu/
mkdir -p /opt/serveraccessd/
cp ./users.db /opt/serveraccessd/server-accessd.db

sed -i 's/^\(passwd:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(group:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(shadow:\s\+\)/\1flantauth /' /etc/nsswitch.conf

getent passwd | grep "vasya:x:3000:3001" || (echo "\033[0;31m$image PASSWD FAILURE\033[0m" && exit 1);
getent passwd | grep "tanya:x:3001:3002" || (echo "\033[0;31m$image PASSWD FAILURE\033[0m" && exit 1);
getent group | grep "admin::3001" || (echo "\033[0;31m$image GROUP FAILURE\033[0m" && exit 1);
getent group | grep "manager::3002" || (echo "\033[0;31m$image GROUP FAILURE\033[0m" && exit 1);
getent shadow | grep "vasya" || (echo "\033[0;31m$image SHADOW FAILURE\033[0m" && exit 1);
getent shadow | grep "tanya" || (echo "\033[0;31m$image SHADOW FAILURE\033[0m" && exit 1);

id vasya && id tanya && echo "\033[0;32m$image SUCCESS \033[0m" && exit 0;

ldd ./out/libnss_flantauth.so.2
echo "\033[0;31m$image FAILURE\033[0m"

EOF

done

# Run in CentOS.
for image in $centosList ; do
  echo Run example in $image
  cat <<EOF | docker run \
    -w /example \
    -v "$SCRIPTDIR":/example \
    --rm -i $image sh -s $image

image=$1

export TERM=xterm-color

cp ./out/libnss_flantauth.so.2 /usr/lib64/
mkdir -p /opt/serveraccessd/
cp ./users.db /opt/serveraccessd/server-accessd.db

sed -i 's/^\(passwd:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(group:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(shadow:\s\+\)/\1flantauth /' /etc/nsswitch.conf

getent passwd | grep "vasya:x:3000:3001" || (echo -e "\033[0;31m$image PASSWD FAILURE\033[0m" && exit 1);
getent passwd | grep "tanya:x:3001:3002" || (echo -e "\033[0;31m$image PASSWD FAILURE\033[0m" && exit 1);
getent group | grep "admin::3001" || (echo -e "\033[0;31m$image GROUP FAILURE\033[0m" && exit 1);
getent group | grep "manager::3002" || (echo -e "\033[0;31m$image GROUP FAILURE\033[0m" && exit 1);
getent shadow | grep "vasya" || (echo -e "\033[0;31m$image SHADOW FAILURE\033[0m" && exit 1);
getent shadow | grep "tanya" || (echo -e "\033[0;31m$image SHADOW FAILURE\033[0m" && exit 1);

id vasya && id tanya && echo -e "\033[0;32m$image SUCCESS \033[0m" && exit 0;

ldd ./out/libnss_flantauth.so.2
echo -e "\033[0;31m$image FAILURE\033[0m"

EOF

done
rm -rf "$SCRIPTDIR"/out