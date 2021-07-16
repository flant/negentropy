#!/usr/bin/env bash

. /etc/kafka/scripts/variables.sh

mkdir -p ${KEYSTORE_PATH}

# Create a truststore if not exists.
if [[ ! -f ${KEYSTORE_PATH}/kafka.server.truststore.jks ]]; then
  # Get CA root certificate.
  gcloud privateca roots describe ${KAFKA_CA_NAME} \
    --location ${KAFKA_CA_LOCATION} --pool=${KAFKA_CA_POOL} \
    --format='get(pemCaCertificates)' > ${KEYSTORE_PATH}/ca.crt

  # Generate truststore from the CA.
  keytool -keystore ${KEYSTORE_PATH}/kafka.server.truststore.jks \
    -alias CARoot -import -file ${KEYSTORE_PATH}/ca.crt \
    -noprompt -keypass ${SERVER_KEY_PASS} -storepass ${SERVER_KEY_PASS}
  rm -f ${KEYSTORE_PATH}/ca.crt
fi

if  [[ -f ${KEYSTORE_PATH}/kafka.server.keystore.jks ]]; then
  # Run renew to check certificate is not expired.
  /etc/kafka/scripts/renew-certificate.sh --dry-run &> /dev/null
  if [ $? -eq 0 ]; then
    exit 0
  fi
  # If `renew-certificate.sh` exited with error, seems there is no certificate in the keystore or it expired.
  rm -f ${KEYSTORE_PATH}/kafka.server.keystore.jks
fi

# Create a keystore.
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks \
  -alias localhost -validity ${CERT_VALIDITY_DAYS} -genkey -keyalg RSA \
  -dname "CN=${FQDN}" -storetype pkcs12  -ext SAN=DNS:${FQDN} \
  -noprompt  -keypass ${SERVER_KEY_PASS} -storepass ${SERVER_KEY_PASS}

# Fetch CA from the trust store.
keytool -keystore ${KEYSTORE_PATH}/kafka.server.truststore.jks \
  -alias CARoot -export -rfc -file ${KEYSTORE_PATH}/ca.crt \
  -keypass ${SERVER_KEY_PASS} -storepass ${SERVER_KEY_PASS}

# Generate CSR to the keystore.
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks \
  -alias localhost -certreq -file ${KEYSTORE_PATH}/server.csr \
  -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS}

# Obtain the keystore's certificate signed with the CA.
gcloud privateca certificates create ${HOSTNAME}-${TIMESTAMP} \
  --issuer-pool ${KAFKA_CA_POOL} --issuer-location ${KAFKA_CA_LOCATION} \
  --csr ${KEYSTORE_PATH}/server.csr --cert-output-file ${KEYSTORE_PATH}/server.crt \
  --validity P${CERT_VALIDITY_DAYS}D
rm -f ${KEYSTORE_PATH}/server.csr

# Import the CA into the keystore.
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks \
  -alias CARoot -import -file ${KEYSTORE_PATH}/ca.crt \
  -noprompt -keypass ${SERVER_KEY_PASS}  -storepass ${SERVER_KEY_PASS}
rm -f ${KEYSTORE_PATH}/ca.crt

# Import the keystore's signed certificate into the keystore.
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks \
  -alias localhost -import -file ${KEYSTORE_PATH}/server.crt \
  -noprompt -keypass ${SERVER_KEY_PASS} -storepass ${SERVER_KEY_PASS}
rm -f ${KEYSTORE_PATH}/server.crt
