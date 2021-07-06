set -exu

# Enable community repository
sed -e 's;^#http\(.*\)/v3.13/community;http\1/v3.13/community;g' -i /etc/apk/repositories

apk update
apk add bash ca-certificates wget curl libcap gettext su-exec tzdata gcompat jq

update-ca-certificates
