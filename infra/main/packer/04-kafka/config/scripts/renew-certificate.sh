#!/usr/bin/env bash

. /etc/kafka-variables.sh

if CERT="$(keytool -exportcert -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks -alias localhost --storepass ${SERVER_KEY_PASS} -rfc)"; then
  CERT_EXPIRATION_DATE="$(echo -n "$CERT" | openssl x509 -dates -noout | grep notAfter | sed "s/notAfter=//")"
  # Convert to YYYY-MM-DD hh:mm:ss
  CERT_EXPIRATION_DATE="$(echo "$CERT_EXPIRATION_DATE" | awk '{ printf "%04d-%02d-%02d %3s", $4, (index("JanFebMarAprMayJunJulAugSepOctNovDec",$1)+2)/3, $2, $3}')"
  CERT_EXPIRATION_DATE_TIMESTAMP="$(date +%s --date "${CERT_EXPIRATION_DATE}")"

  if (( ${CERT_EXPIRATION_DATE_TIMESTAMP}-${TIMESTAMP}>${CERT_EXPIRE_SECONDS} )); then
      echo "The certificate is not expired. Exit."
      exit 0
  fi
fi

if [[ "$1" == "--dry-run" ]]; then
  exit 1
fi

LOCKFILE="kafka-renew-certificate.lock"

if [[ -f /tmp/${LOCKFILE} ]]; then
    >&2 echo "Another copy of the script is running locally. Exit."
    exit 1
fi

echo "${NODE_ID}" > /tmp/${LOCKFILE}

while gsutil ls gs://${KAFKA_BUCKET}/${LOCKFILE} &>/dev/null; do
    >&2 echo "Another Kafka instance certificates renew process is running. Sleep for 15 sec."
    sleep 15
done

gsutil cp /tmp/${LOCKFILE} gs://${KAFKA_BUCKET}

rc-service kafka stop
rc-service zookeeper stop

# Remove the keystore. It will be recreated during the following `zookeeper start`.
rm -f ${KEYSTORE_PATH}/kafka.server.keystore.jks

rc-service zookeeper start
rc-service kafka start

gsutil rm gs://${KAFKA_BUCKET}/${LOCKFILE}
rm -f /tmp/${LOCKFILE}
