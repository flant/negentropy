set -exu

apk update
apk add bash ca-certificates wget curl libcap gettext su-exec tzdata gcompat

update-ca-certificates
