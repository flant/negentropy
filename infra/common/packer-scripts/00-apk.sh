set -exu

apk update
apk add bash ca-certificates wget curl libcap gettext

update-ca-certificates
