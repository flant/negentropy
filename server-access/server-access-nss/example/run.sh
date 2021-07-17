#!/bin/sh

ubuntuList=
ubuntuList="ubuntu:16.04 ubuntu:18.04 ubuntu:20.04 ubuntu:21.04"

alpineList=
#alpineList="alpine:3.12 alpine:3.13 alpine:3.14"

debianList=
debianList="debian:7 debian:8 debian:9 debian:10"

centosList=
centosList="centos:6 centos:7 centos:8"

# build universal .so in buster based image.
if [[ ! -f ./out/libnss_flantauth.so.2 ]] ; then
  echo "build libnss_flantauth.so.2 ..."

  cat <<'EOF' | docker run \
    -w /src/server-access-nss \
    -v $(pwd)/../..:/src \
    -v $(pwd)/out:/out \
    --rm -i golang:1.15-stretch sh -

./build.sh
mv libnss_flantauth.so.2 /out
ldd /out/libnss_flantauth.so.2
EOF
else
  echo "libnss_flantauth.so.2 exists, skip building"
fi

# Run in Debian and Ubuntu.
for image in $debianList $ubuntuList ; do
  echo Run example in $image
  cat <<EOF | docker run \
    -w /example \
    -v $(pwd):/example \
    --rm -i $image sh -s $image

image=$1

cp ./out/libnss_flantauth.so.2 /lib/x86_64-linux-gnu/
mkdir -p /opt/negentropy
cp ./users.db /opt/negentropy/server-access.db

sed -i 's/^\(passwd:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(group:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(shadow:\s\+\)/\1flantauth /' /etc/nsswitch.conf

id vasya && id tanya && echo "\033[0;32m$image SUCCESS \033[0m" && exit

ldd ./out/libnss_flantauth.so.2
echo "\033[0;31m$image FAILURE\033[0m"

EOF

done

# Run in CentOS.
for image in $centosList ; do
  echo Run example in $image
  cat <<EOF | docker run \
    -w /example \
    -v $(pwd):/example \
    --rm -i $image sh -s $image

image=$1

cp ./out/libnss_flantauth.so.2 /usr/lib64/
mkdir -p /opt/negentropy
cp ./users.db /opt/negentropy/server-access.db

sed -i 's/^\(passwd:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(group:\s\+\)/\1flantauth /' /etc/nsswitch.conf
sed -i 's/^\(shadow:\s\+\)/\1flantauth /' /etc/nsswitch.conf

id vasya && id tanya && echo -e "\033[0;32m$image SUCCESS \033[0m" && exit

ldd ./out/libnss_flantauth.so.2
echo -e "\033[0;31m$image FAILURE\033[0m"

EOF

done
