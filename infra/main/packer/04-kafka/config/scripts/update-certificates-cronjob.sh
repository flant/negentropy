#!/usr/bin/env bash
. /etc/kafka/scripts/variables.sh

set -Eeo pipefail
LOCKFILE="kafka-update-certificates.lock"

CERT_EXPIRATION_DATE="$(keytool -exportcert -keystore /data/keystore/kafka.server.keystore.jks --storepass ${SERVER_KEY_PASS} -rfc | openssl x509 -dates -noout | grep notAfter | sed "s/notAfter=//")"
# Convert to YYYY-MM-DD hh:mm:ss
CERT_EXPIRATION_DATE="$(echo "$CERT_EXPIRATION_DATE" | awk '{ printf "%04d-%02d-%02d %3s", $4, (index("JanFebMarAprMayJunJulAugSepOctNovDec",$1)+2)/3, $2, $3}')"
CERT_EXPIRATION_DATE_TIMESTAMP="$(date +%s --date "${CERT_EXPIRATION_DATE}")"

if (( ${CERT_EXPIRATION_DATE_TIMESTAMP}-${TIMESTAMP}>${CERT_EXPIRE_SECONDS} )); then
    echo "Cert not expired. Exiting."
    exit 0
fi

if [[ -f /tmp/${LOCKFILE} ]]; then
    >&2 echo "Another copy of script is running local. Exiting." 
    exit 0
fi

echo "${NODE_ID}" > /tmp/${LOCKFILE}

while gsutil ls gs://${KAFKA_BUCKET}/${LOCKFILE} 1>/dev/null 2>/dev/null; do
    echo "Another kafka certificates update process is working. Sleeping 15sec."
    sleep 15
done

gsutil cp /tmp/${LOCKFILE} gs://${KAFKA_BUCKET}

# TODO: If following will be failing continuously no other kafkas would be able to update their certificates. Because gsutil rm won't run.
rc-service kafka stop
rc-service zookeeper stop

./generate_keystores.sh

rc-service zookeeper start
rc-service kafka start

gsutil rm gs://${KAFKA_BUCKET}/${LOCKFILE}
rm -f /tmp/${LOCKFILE}
