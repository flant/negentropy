set -exu

apk update
apk add certbot nginx

apk add py3-pip
pip3 install certbot-dns-google

mkdir -p /tmp/letsencrypt
ln -s /tmp/letsencrypt /etc/letsencrypt

cat <<'EOF' > /etc/nginx-config.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

mkdir -p /var/lib/nginx/logs
chown -R nginx:nginx /var/lib/nginx

mkdir -p /tmp/nginx
envsubst '$INTERNAL_ADDRESS,$VAULT_INTERNAL_FQDN,$VAULT_INTERNAL_DOMAIN' < /etc/nginx/nginx-vault-internal.conf > /tmp/nginx/nginx-vault-internal.conf
envsubst '$INTERNAL_ADDRESS,$VAULT_PUBLIC_FQDN,$VAULT_PUBLIC_DOMAIN' < /etc/nginx/nginx-vault-public.conf > /tmp/nginx/nginx-vault-public.conf
EOF

chmod +x /etc/nginx-config.sh


cat <<'EOF' > /etc/nginx-public-cert.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

NGINX_LE_ARCHIVE="$(hostname)-le-public-cert.tar.gz"

upload() {
  pushd /tmp
  tar -czvf ${NGINX_LE_ARCHIVE} letsencrypt
  gsutil cp ${NGINX_LE_ARCHIVE} gs://${TFSTATE_BUCKET}/
  popd
}

renew() {
  CHECKSUM_BEFORE="$(find /tmp/letsencrypt/archive/ -type f \( -exec sha1sum {} \; \) | awk '{print $1}' | sort | sha1sum | awk '{print $1}')"
  certbot renew --quiet
  CHECKSUM_AFTER="$(find /tmp/letsencrypt/archive/ -type f \( -exec sha1sum {} \; \) | awk '{print $1}' | sort | sha1sum | awk '{print $1}')"
  if [[ "$CHECKSUM_BEFORE" != "$CHECKSUM_AFTER" ]]; then
      /etc/init.d/nginx reload
      upload
  fi
}

if [ -d "/tmp/letsencrypt" ]; then
  renew
  exit 0
fi

mkdir -p /tmp/letsencrypt
pushd /tmp
gsutil cp gs://${TFSTATE_BUCKET}/${NGINX_LE_ARCHIVE} .
if [ -f "${NGINX_LE_ARCHIVE}" ]; then
  tar xvf ${NGINX_LE_ARCHIVE}
else
  certbot certonly --dns-google -d ${VAULT_PUBLIC_FQDN} --non-interactive --agree-tos -m ${VAULT_AUTH_PUBLIC_CERTIFICATE_EMAIL}
  openssl dhparam -out /tmp/letsencrypt/ssl-dhparams.pem 2048
  upload
fi
popd
EOF

chmod +x /etc/nginx-public-cert.sh

# Following init.d script almost copied as is, except of:
## 1. In `start_pre` we need to ensure certificates are exists and fresh by executing /etc/nginx-public-cert.sh.
## 2. In `start_pre` we need to substitute nginx virtual hosts configuration.
## 3. In `depend` we should start after the vault, because we are using its certificates.
cat <<'EOF' > /etc/init.d/nginx
#!/sbin/openrc-run

description="Nginx http and reverse proxy server"
extra_commands="checkconfig"
extra_started_commands="reload reopen upgrade"

cfgfile=${cfgfile:-/etc/nginx/nginx.conf}
pidfile=/run/nginx/nginx.pid
command=${command:-/usr/sbin/nginx}
command_args="-c $cfgfile"
required_files="$cfgfile"

depend() {
	need net
	use dns logger netmount
	after vault
}

# TODO: remove debug output forwarding.
start_pre() {
	checkpath --directory --owner nginx:nginx ${pidfile%/*} \
	  && /etc/nginx-public-cert.sh &> /var/log/nginx-public-cert.log \
	  && /etc/nginx-config.sh
	$command $command_args -t -q
}

checkconfig() {
	ebegin "Checking $RC_SVCNAME configuration"
	start_pre
	eend $?
}

reload() {
	ebegin "Reloading $RC_SVCNAME configuration"
	start_pre && start-stop-daemon --signal HUP --pidfile $pidfile
	eend $?
}

reopen() {
	ebegin "Reopening $RC_SVCNAME log files"
	start-stop-daemon --signal USR1 --pidfile $pidfile
	eend $?
}

upgrade() {
	start_pre || return 1

	ebegin "Upgrading $RC_SVCNAME binary"

	einfo "Sending USR2 to old binary"
	start-stop-daemon --signal USR2 --pidfile $pidfile

	einfo "Sleeping 3 seconds before pid-files checking"
	sleep 3

	if [ ! -f $pidfile.oldbin ]; then
		eerror "File with old pid ($pidfile.oldbin) not found"
		return 1
	fi

	if [ ! -f $pidfile ]; then
		eerror "New binary failed to start"
		return 1
	fi

	einfo "Sleeping 3 seconds before WINCH"
	sleep 3 ; start-stop-daemon --signal 28 --pidfile $pidfile.oldbin

	einfo "Sending QUIT to old binary"
	start-stop-daemon --signal QUIT --pidfile $pidfile.oldbin

	einfo "Upgrade completed"

	eend $? "Upgrade failed"
}
EOF

rc-update add nginx

# The same script renews certificates.
echo "0 12 * * * /etc/nginx-public-cert.sh" >> /etc/crontabs/root
