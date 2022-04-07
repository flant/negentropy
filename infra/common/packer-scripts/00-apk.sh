set -exu

# Enable community repository
sed -e 's;^#*.http\(.*\)/v3.15/community;http\1/v3.15/community;g' -i /etc/apk/repositories

apk update
apk add bash ca-certificates wget curl libcap gettext su-exec tzdata gcompat cloud-init e2fsprogs-extra

update-ca-certificates
