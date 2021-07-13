set -exu

apk update
apk add certbot nginx

apk add py3-pip
pip3 install certbot-dns-google

mkdir -p /tmp/letsencrypt
ln -s /tmp/letsencrypt /etc/letsencrypt

mkdir -p /tmp/letsencrypt
# if no archive and /tmp/letsencrypt is empty - generate cert, else run renew
# TODO: remove --staging
certbot certonly --dns-google -d ew1a1.auth.negentropy.p.golovin.hf.flant.com --staging --non-interactive --agree-tos -m pavel.golovin@flant.com
openssl dhparam -out /tmp/letsencrypt/ssl-dhparams.pem 2048
# call renew script here to upload certs to the vault bucket

# make one more variable to have external root domain
# covert ew1a1.auth.negentropy.flant.local to .com and one more auth.negentropy.flant.com

# add to vault cert additional domain auth.negentropy.flant.local

mkdir /tmp/nginx
envsubst '$INTERNAL_ADDRESS' < /etc/nginx/http.d/nginx-vault-public.conf > /tmp/nginx/nginx-vault-public.conf

envsubst '$INTERNAL_ADDRESS' nginx-vault-internal.conf > /tmp/nginx/
envsubst '$INTERNAL_ADDRESS' nginx-vault-public.conf > /tmp/nginx/

# cron
# todo: we need special renew script with cert upload to bucket on renewal
echo "0 12 * * * /usr/bin/certbot renew --quiet" >> /etc/crontabs/root
