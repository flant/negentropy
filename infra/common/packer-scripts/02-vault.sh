set -exu

# We are building our own vault binary and packer should upload it before running this script.
chmod +x /bin/vault

addgroup vault && \
adduser -S -G vault vault

cat <<'EOF' > /etc/vault-config.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

envsubst < /etc/vault.hcl > /tmp/vault.hcl
EOF

chmod +x /etc/vault-config.sh

cat <<'EOF' > /etc/get-ca.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

# TODO: add check for empty argument and if it empty then show help output with available parameters.
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --get) target="get";;
        --update) target="update";;
        *) echo "Unknown parameter $1"; exit 1;;
    esac
    shift
done

# TODO: make a symlink of /usr/local/share/ca-certificates to /tmp/ca-certificates in base image and use tmp folder here
if [ "$target" == "get" ]; then
    if [ -f "/usr/local/share/ca-certificates/ca.crt" ]; then
        echo "CA already exists"
    else
        echo "Get CA"
        while true; do
            gcloud privateca roots describe ${VAULT_CA_NAME} --location=${VAULT_CA_LOCATION} --pool=${VAULT_CA_POOL} --format="get(pemCaCertificates)" > /usr/local/share/ca-certificates/ca.crt
            status=$?
            if [ $status -eq 0 ]; then
                break
            fi
        done
        update-ca-certificates &> /dev/null
    fi
fi

# TODO: add support for check and update CA, when it expire soon.
if [ "$target" == "update" ]; then
    echo "Update CA not implemented yet :("
fi
EOF

chmod +x /etc/get-ca.sh

cat <<'EOF' > /etc/get-cert.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

getcert() {
pushd /tmp
echo "Generate CSR"
openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj "/C=RU/O=JSC Flant/CN=${VAULT_INTERNAL_FQDN}" -addext "subjectAltName=DNS:${VAULT_INTERNAL_FQDN},IP:${INTERNAL_ADDRESS},IP:127.0.0.1"
echo "Signing certificate"
while true; do
    gcloud privateca certificates create vault-conf --issuer-pool ${VAULT_CA_POOL} --issuer-location ${VAULT_CA_LOCATION} --csr server.csr --cert-output-file server.crt --validity "P${VAULT_CERT_VALIDITY_DAYS}D"
    status=$?
    if [ $status -eq 0 ]; then
        break
    fi
done
echo "Move certificates and set right permissions"
mv server.crt vault.crt
mv server.key vault.key
chown vault:vault vault.crt
chown vault:vault vault.key
echo "Cleanup"
rm server.csr
popd
}

# TODO: add check for empty argument and if it empty then show help output with available parameters.
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --get) target="get";;
        --update) target="update";;
        *) echo "Unknown parameter $1"; exit 1;;
    esac
    shift
done

if [ "$target" == "get" ]; then
    if [[ -f "/tmp/vault.crt" && -f "/tmp/vault.key" ]]; then
        echo "Vault certificate already exists"
    else
        getcert
    fi
fi

if [ "$target" == "update" ]; then
    CERT_EXPIRE_SECONDS="${VAULT_CERT_EXPIRE_SECONDS}"
    TIMESTAMP="$(date +"%s")"
    CERT_EXPIRATION_DATE="$(openssl x509 -enddate -noout -in /tmp/vault.crt | sed "s/notAfter=//")"
    # convert to YYYY-MM-DD hh:mm:ss into the same variable
    CERT_EXPIRATION_DATE="$(echo "$CERT_EXPIRATION_DATE" | awk '{ printf "%04d-%02d-%02d %3s", $4, (index("JanFebMarAprMayJunJulAugSepOctNovDec",$1)+2)/3, $2, $3}')"
    CERT_EXPIRATION_DATE_TIMESTAMP="$(date +%s --date "${CERT_EXPIRATION_DATE}")"

    if (( ${CERT_EXPIRATION_DATE_TIMESTAMP}-${TIMESTAMP}>${CERT_EXPIRE_SECONDS} )); then
        echo "Vault certificate not expired. Exiting."
        exit 0
    else
        echo "Vault certificate expiring"
        getcert
        echo "Vault certificate updated"
        /etc/init.d/vault reload
        echo "Vault reloaded"
    fi
fi
EOF

chmod +x /etc/get-cert.sh

cat <<'EOF' > /etc/vault-init.sh
#!/usr/bin/env bash

. /etc/vault-variables.sh

while true; do
    vault status &>/dev/null
    status=$?
    if [ $status -eq 0 ]; then
      echo "Vault already initialized."
      exit 0
    elif [ $status -eq 2 ]; then
      break
    fi
    echo "Waiting for vault server start, sleeping for 2 seconds..."
    sleep 2
done

echo "Starting vault initialization."
pushd /tmp
until [ -f "${VAULT_ROOT_TOKEN_PGP_KEY}" ]; do
  echo "Retrieving vault-root-token pgp key"
  gsutil cp gs://${TFSTATE_BUCKET}/${VAULT_ROOT_TOKEN_PGP_KEY} .
  sleep 2
done
vault_recovery_pgp_keys="$(find "/etc/recovery-pgp-keys/" -type f | tr '\n' ',' | sed 's/,$//g')"
vault_init_out="$(vault operator init -root-token-pgp-key="${VAULT_ROOT_TOKEN_PGP_KEY}" -recovery-shares="${VAULT_RECOVERY_SHARES}" -recovery-threshold="${VAULT_RECOVERY_THRESHOLD}" -recovery-pgp-keys="${vault_recovery_pgp_keys}")"
echo "$vault_init_out" | grep "^Initial Root Token: " | awk '{print $4}' > "${VAULT_ROOT_TOKEN_ENCRYPTED}"
echo "$vault_init_out" | grep "^Recovery Key " | awk '{print $4}' > "${VAULT_RECOVERY_KEYS_ENCRYPTED}"
gsutil cp {"${VAULT_ROOT_TOKEN_ENCRYPTED}","${VAULT_RECOVERY_KEYS_ENCRYPTED}"} gs://${TFSTATE_BUCKET}/
rm -rf "${VAULT_ROOT_TOKEN_ENCRYPTED}"
rm -rf "${VAULT_RECOVERY_KEYS_ENCRYPTED}"
rm -rf "${VAULT_ROOT_TOKEN_PGP_KEY}"
popd
EOF

chmod +x /etc/vault-init.sh

cat <<'EOF' > /etc/init.d/vault
#!/sbin/openrc-run
name="Vault server"
description="Vault is a tool for securely accessing secrets"
description_reload="Reload configuration"

extra_started_commands="reload"

command="/bin/vault"
command_args="server -config /tmp/vault.hcl"
command_user="vault:vault"

supervisor=supervise-daemon
output_log="/var/log/vault.log"
error_log="/var/log/vault.log"
respawn_max=0
respawn_delay=10

depend() {
	need net
	after firewall
}

# TODO: remove debug output forwarding.
start_pre() {
	checkpath -f -m 0644 -o "$command_user" "$output_log" "$error_log" \
    && /bin/update-hostname \
    && /etc/get-ca.sh --get &> /var/log/vault-ca.log \
    && /etc/get-cert.sh --get &> /var/log/vault-cert.log \
    && /etc/vault-config.sh
}

start_post() {
  /etc/vault-init.sh &> /var/log/vault-init.log
}

reload() {
	start_pre \
		&& ebegin "Reloading $RC_SVCNAME configuration" \
		&& $supervisor "$RC_SVCNAME" --signal HUP
	eend $?
}
EOF

chmod +x /etc/init.d/vault
rc-update add vault

# Add update certificate cronjob
echo '*/30 * * * * /etc/get-cert.sh --update' >> /etc/crontabs/root

# Add memlock capability
setcap cap_ipc_lock=+ep /bin/vault

# Increase ulimit
cat <<'EOF' > /etc/conf.d/vault
rc_ulimit="-n 65536"
EOF

echo "source /etc/vault-variables.sh" >> /root/.profile
